package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/compute"
	"github.com/pulumi/pulumi-local/sdk/go/local"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createAnsibleInventory(ctx *pulumi.Context, filePath string, masterInstance *compute.Instance, replicaInstance *compute.Instance, privateKeyPath string) error {
	masterIp := masterInstance.NetworkInterfaces.Index(pulumi.Int(0)).
		AccessConfigs().Index(pulumi.Int(0)).
		NatIp()
	replicaIp := replicaInstance.NetworkInterfaces.Index(pulumi.Int(0)).
		AccessConfigs().Index(pulumi.Int(0)).
		NatIp()
	inventoryContent := pulumi.All(masterIp, replicaIp).ApplyT(func(ips []interface{}) string {
		return "[postgres]\n" +
			"master ansible_host=" + *ips[0].(*string) + "\n" +
			"replica ansible_host=" + *ips[1].(*string) + "\n\n" +
			"[postgres:vars]\n" +
			"ansible_user=user0\n" +
			"ansible_ssh_private_key_file=" + privateKeyPath + "\n"
	}).(pulumi.StringOutput)

	// Write the inventory file using the local provider
	_, err := local.NewFile(ctx, "inventoryFile", &local.FileArgs{
		Content:  inventoryContent,
		Filename: pulumi.String(filePath),
	})
	if err != nil {
		return err
	}
	return nil
}
