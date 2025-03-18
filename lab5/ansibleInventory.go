package main

import (
	"fmt"

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
	masterLocalIp := masterInstance.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp()
	inventoryContent := pulumi.All(masterIp, masterLocalIp, replicaIp).ApplyT(func(ips []interface{}) string {

		content := "[postgres_master]\n" +
			"master ansible_host=" + *ips[0].(*string) + "\n" +
			"[postgres_master:vars]\n" +
			"ansible_user=user0\n" +
			"ansible_ssh_private_key_file=" + privateKeyPath + "\n" +
			"[postgres_replica]\n" +
			"replica1 ansible_host=" + *ips[2].(*string) + "\n" +
			"[postgres_replica:vars]\n" +
			"ansible_user=user0\n" +
			"master_ip=" + *ips[1].(*string) + "\n" +
			"ansible_ssh_private_key_file=" + privateKeyPath + "\n"
		fmt.Println("content", content)
		return content
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
