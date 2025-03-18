package main

import (
	"strings"

	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const masterStartupScript = `#!/bin/bash
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib

# Enable listening on all interfaces
sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/g" /etc/postgresql/14/main/postgresql.conf

# Allow replication connections from any IP (for demo purposes; restrict in production)
echo "host replication all 0.0.0.0/0 md5" | sudo tee -a /etc/postgresql/14/main/pg_hba.conf
echo "host postgres all 0.0.0.0/0 md5" | sudo tee -a /etc/postgresql/14/main/pg_hba.conf
echo "wal_level = logical" | sudo tee -a /etc/postgresql/14/main/postgresql.conf

# Restart PostgreSQL to apply changes
sudo systemctl restart postgresql

# Create a replication role with replication privileges
sudo -u postgres psql -c "CREATE ROLE replicator WITH REPLICATION PASSWORD 'replicator_pass' LOGIN;"
`

func NewPostgresMaster(ctx *pulumi.Context, provider pulumi.ProviderResource, zone string, sshPublicKeys []string) (*compute.Instance, error) {
	// Define the startup script for the PostgreSQL master instance.
	return compute.NewInstance(ctx, "postgres-master", &compute.InstanceArgs{
		Name:        pulumi.String("postgres-master"),
		MachineType: pulumi.String("e2-micro"),
		Zone:        pulumi.String(zone),
		BootDisk: &compute.InstanceBootDiskArgs{
			InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
				Image: pulumi.String("ubuntu-os-cloud/ubuntu-2204-lts"),
				Size:  pulumi.Int(10),
			},
		},
		Tags: pulumi.StringArray{
			pulumi.String("postgres"),
			pulumi.String("allow-ssh"),
		},
		NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
			&compute.InstanceNetworkInterfaceArgs{
				Network: pulumi.String("default"),
				AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
					&compute.InstanceNetworkInterfaceAccessConfigArgs{},
				},
			},
		},
		Metadata: pulumi.StringMap{
			"startup-script": pulumi.String(masterStartupScript),
			"ssh-keys":       pulumi.String(strings.Join(sshPublicKeys, "\n")),
		},
	}, pulumi.Provider(provider))
}
func newReplicaStartupScript(masterIP pulumi.StringOutput) pulumi.StringOutput {
	return pulumi.Sprintf(`#!/bin/bash
	MASTER_IP="%s"
	
	sudo apt-get update
	sudo apt-get install -y postgresql postgresql-contrib
	
	# Stop PostgreSQL so we can set up replication
	sudo systemctl stop postgresql
	
	# Clear any existing data in the data directory
	sudo rm -rf /var/lib/postgresql/14/main
	
	# Set the replication password environment variable for pg_basebackup
	export PGPASSWORD="replicator_pass"
	
	# Perform a base backup from the master server
	sudo -u postgres env PGPASSWORD='replicator_pass' pg_basebackup  --host="$MASTER_IP" --username=replicator -P --wal-method=stream --pgdata=/var/lib/postgresql/14/main
	
	
	# Start PostgreSQL service
	sudo systemctl start postgresql
	`, masterIP)

}

func NewPostgresReplica(ctx *pulumi.Context, provider pulumi.ProviderResource, zone string, masterIP pulumi.StringOutput, sshPublicKeys []string) (*compute.Instance, error) {
	replicaStartupScript := newReplicaStartupScript(masterIP)
	return compute.NewInstance(ctx, "postgres-replica", &compute.InstanceArgs{
		Name:        pulumi.String("postgres-replica"),
		MachineType: pulumi.String("e2-micro"),
		Zone:        pulumi.String(zone),
		BootDisk: &compute.InstanceBootDiskArgs{
			InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
				Image: pulumi.String("ubuntu-os-cloud/ubuntu-2204-lts"),
				Size:  pulumi.Int(10),
			},
		},
		NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
			&compute.InstanceNetworkInterfaceArgs{
				Network: pulumi.String("default"),
				AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
					&compute.InstanceNetworkInterfaceAccessConfigArgs{},
				},
			},
		},
		Tags: pulumi.StringArray{
			pulumi.String("postgres"),
			pulumi.String("allow-ssh"),
		},
		Metadata: pulumi.StringMap{
			"startup-script": replicaStartupScript,
			"ssh-keys":       pulumi.String(strings.Join(sshPublicKeys, "\n")),
		},
	}, pulumi.Provider(provider))
}
