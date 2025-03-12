package main

import (
	"strings"

	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Read configuration values
		cfg := config.New(ctx, "")
		project := cfg.Require("project")
		region := cfg.Get("region")
		if region == "" {
			region = "us-central1"
		}
		zone := cfg.Get("zone")
		if zone == "" {
			zone = "us-central1-a"
		}

		keys, err := keysFromFiles(strings.Split(cfg.Get("sshPublicKeys"), ",")...)
		if err != nil {
			return err
		}
		gcpProvider, err := gcp.NewProvider(ctx, "gcp-provider", &gcp.ProviderArgs{
			Project: pulumi.String(project),
			Region:  pulumi.String(region),
		})
		if err != nil {
			return err
		}

		_, err = compute.NewFirewall(ctx, "allow-ssh", &compute.FirewallArgs{
			Name:    pulumi.String("allow-ssh"),
			Network: pulumi.String("default"),
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String("22"),
					},
				},
			},
			SourceRanges: pulumi.StringArray{
				pulumi.String("0.0.0.0/0"),
			},
		}, pulumi.Provider(gcpProvider))
		if err != nil {
			return err
		}

		postgresMaster, err := NewPostgresMaster(ctx, gcpProvider, zone, keys)
		if err != nil {
			return err
		}

		// Retrieve the master instance's internal IP address.
		masterIP := postgresMaster.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp()

		postgresReplica, err := NewPostgresReplica(ctx, gcpProvider, zone, masterIP.Elem(), keys)
		if err != nil {
			return err
		}

		replicaIP := postgresReplica.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp().Elem()

		// Compute a string array containing the allowed source IPs in CIDR /32 notation.
		allowedIPs := pulumi.All(masterIP.Elem(), replicaIP).ApplyT(func(ips []interface{}) []string {
			m := ips[0].(string)
			r := ips[1].(string)
			return []string{
				m + "/32",
				r + "/32",
			}
		}).(pulumi.StringArrayOutput)

		_, err = compute.NewFirewall(ctx, "allow-postgres-ports", &compute.FirewallArgs{
			Name:    pulumi.String("allow-postgres-ports"),
			Network: pulumi.String("default"),
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String("5432"),
						pulumi.String("5433"),
						pulumi.String("5434"),
						pulumi.String("5435"),
					},
				},
			},
			SourceRanges: allowedIPs,
		}, pulumi.Provider(gcpProvider))
		if err != nil {
			return err
		}
		ctx.Export("postgresMasterExternalIP",
			postgresMaster.NetworkInterfaces.Index(pulumi.Int(0)).
				AccessConfigs().Index(pulumi.Int(0)).
				NatIp())
		ctx.Export("postgresReplicaExternalIP",
			postgresReplica.NetworkInterfaces.Index(pulumi.Int(0)).
				AccessConfigs().Index(pulumi.Int(0)).
				NatIp())
		return nil
	})
}
