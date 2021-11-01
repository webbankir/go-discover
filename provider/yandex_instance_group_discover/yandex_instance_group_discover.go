package yandex_instance_group_discover
import (
	"context"
	"fmt"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1/instancegroup"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"io/ioutil"
	"log"
	"strings"
)

type Provider struct{}

// Help returns help information for the Yandex.Cloud package.
func (p *Provider) Help() string {
	return `Yandex Cloud:
    provider:          "yandex-cloud"
    folder_id:         FolderID.
    metadata_key:      metadata_key.
    metadata_value:    metadata_value.
	cidr:			   cidr
`
}

func (p *Provider) Addrs(args map[string]string, l *log.Logger) ([]string, error) {
	if l == nil {
		l = log.New(ioutil.Discard, "", 0)
	}

	var ipAddresses []string
	var yandexCredentials ycsdk.Credentials

	serviceAccountKeyFile := args["service_account_key"]
	if serviceAccountKeyFile == "" {
		serviceAccountKeyFile = "/etc/yandex_service_account_key.json"
	}


	providerConfiguration := YandexCloudInstanceGroupProviderConfiguration{
		FolderID:              args["folder_id"],
		InstanceGroupName:     args["instance_group_name"],
		ServiceAccountKeyFile: serviceAccountKeyFile,
	}

	// At First read service account key file
	serviceAccountKey, err := iamkey.ReadFromJSONFile(providerConfiguration.ServiceAccountKeyFile)
	if err == nil {
		yandexCredentials, err = ycsdk.ServiceAccountKey(serviceAccountKey)
	}

	if yandexCredentials == nil {
		oauthToken := ""
		oauthTokenBytes, err := ioutil.ReadFile("/etc/yandex_oauth_token")
		if err == nil {
			oauthToken = strings.TrimSuffix(string(oauthTokenBytes), "\n")
		} else {
			return nil, err
		}
		yandexCredentials = ycsdk.OAuthToken(oauthToken)
	}

	ctx := context.Background()

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: yandexCredentials,
	})

	if err != nil {
		return nil, fmt.Errorf("discover-yandex-cloud-instance-group: Got error while connecting to Yandex.Cloud: %v", err)
	}

	groupsRequest := instancegroup.ListInstanceGroupsRequest{
		Filter:   fmt.Sprintf(`%s = "%s"`, "name", providerConfiguration.InstanceGroupName),
		FolderId: providerConfiguration.FolderID,
	}

	listOfGroups, err := sdk.InstanceGroup().InstanceGroup().List(ctx, &groupsRequest)

	if err != nil {
		return nil, fmt.Errorf("discover-yandex-cloud-instance-group: Got error while getting list of Instance Groups from Yandex.Cloud: %v", err)
	}

	for _, group := range listOfGroups.InstanceGroups {
		req := instancegroup.ListInstanceGroupInstancesRequest{
			InstanceGroupId: group.Id,
			PageSize:        1000,
		}
		listOfInstances, err := sdk.InstanceGroup().InstanceGroup().ListInstances(ctx, &req)

		if err != nil {
			return nil, fmt.Errorf("discover-yandex-cloud-instance-group: Got error while getting list of instances in group from Yandex.Cloud: %v", err)
		}

		for _, instance := range listOfInstances.Instances {
			ipAddress := instance.NetworkInterfaces[0]
			if ipAddress != nil {
				ipAddresses = append(ipAddresses, ipAddress.PrimaryV4Address.Address)
			}
		}
	}

	return ipAddresses, nil
}

type YandexCloudInstanceGroupProviderConfiguration struct {
	ServiceAccountKeyFile string
	FolderID              string
	InstanceGroupName     string
}