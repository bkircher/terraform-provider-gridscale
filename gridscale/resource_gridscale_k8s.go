package gridscale

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gridscale/gsclient-go/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	errHandler "github.com/terraform-providers/terraform-provider-gridscale/gridscale/error-handler"

	"log"
)

const k8sTemplateCategoryName = "kubernetes"

func resourceGridscaleK8s() *schema.Resource {
	return &schema.Resource{
		Create: resourceGridscaleK8sCreate,
		Read:   resourceGridscaleK8sRead,
		Delete: resourceGridscaleK8sDelete,
		Update: resourceGridscaleK8sUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The human-readable name of the object. It supports the full UTF-8 character set, with a maximum of 64 characters",
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"kubeconfig": {
				Type:        schema.TypeString,
				Description: "K8s config data",
				Computed:    true,
			},
			"listen_port": {
				Type:        schema.TypeSet,
				Description: "Ports that PaaS service listens to",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"security_zone_uuid": {
				Type:        schema.TypeString,
				Description: "Security zone UUID linked to PaaS service",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			"network_uuid": {
				Type:        schema.TypeString,
				Description: "Network UUID containing security zone",
				Computed:    true,
			},
			"k8s_release": {
				Type:         schema.TypeString,
				Description:  "Release number of k8s service",
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"k8s_release_computed": {
				Type:        schema.TypeString,
				Description: "Release number of k8s service. The `k8s_release_computed` will be different from `k8s_release`, when `k8s_release` is updated outside of terraform.",
				Computed:    true,
			},
			"worker_node_ram": {
				Type:        schema.TypeInt,
				Description: "Memory per worker node",
				Optional:    true,
				Default:     16,
			},
			"worker_node_cores": {
				Type:        schema.TypeInt,
				Description: "Cores per worker node",
				Optional:    true,
				Default:     4,
			},
			"worker_node_count": {
				Type:        schema.TypeInt,
				Description: "Number of worker nodes",
				Optional:    true,
				Default:     3,
			},
			"worker_node_storage": {
				Type:        schema.TypeInt,
				Description: "Storage (in GiB) per worker node",
				Optional:    true,
				Default:     30,
			},
			"worker_node_storage_type": {
				Type:        schema.TypeString,
				Description: "Storage type (one of storage, storage_high, storage_insane)",
				Optional:    true,
				Default:     "storage_insane",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					valid := false
					for _, stype := range storageTypes {
						if v.(string) == stype {
							valid = true
							break
						}
					}
					if !valid {
						errors = append(errors, fmt.Errorf("%v is not a valid storage type. Valid types are: %v", v.(string), strings.Join(storageTypes, ",")))
					}
					return
				},
			},
			"usage_in_minute": {
				Type:        schema.TypeInt,
				Description: "Number of minutes that PaaS service is in use",
				Computed:    true,
			},
			"current_price": {
				Type:        schema.TypeFloat,
				Description: "Current price of PaaS service",
				Computed:    true,
			},
			"change_time": {
				Type:        schema.TypeString,
				Description: "Time of the last change",
				Computed:    true,
			},
			"create_time": {
				Type:        schema.TypeString,
				Description: "Time of the creation",
				Computed:    true,
			},
			"status": {
				Type:        schema.TypeString,
				Description: "Current status of PaaS service",
				Computed:    true,
			},
			"labels": {
				Type:        schema.TypeSet,
				Description: "List of labels.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Minute),
			Update: schema.DefaultTimeout(15 * time.Minute),
			Delete: schema.DefaultTimeout(15 * time.Minute),
		},
	}
}

func resourceGridscaleK8sRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gsclient.Client)
	errorPrefix := fmt.Sprintf("read k8s (%s) resource -", d.Id())
	paas, err := client.GetPaaSService(context.Background(), d.Id())
	if err != nil {
		if requestError, ok := err.(gsclient.RequestError); ok {
			if requestError.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}
	props := paas.Properties
	creds := props.Credentials
	if err = d.Set("name", props.Name); err != nil {
		return fmt.Errorf("%s error setting name: %v", errorPrefix, err)
	}
	if creds != nil && len(creds) > 0 {
		if err = d.Set("kubeconfig", creds[0].KubeConfig); err != nil {
			return fmt.Errorf("%s error setting kubeconfig: %v", errorPrefix, err)
		}
	}
	if err = d.Set("security_zone_uuid", props.SecurityZoneUUID); err != nil {
		return fmt.Errorf("%s error setting security_zone_uuid: %v", errorPrefix, err)
	}

	if err = d.Set("usage_in_minute", props.UsageInMinutes); err != nil {
		return fmt.Errorf("%s error setting usage_in_minute: %v", errorPrefix, err)
	}
	if err = d.Set("current_price", props.CurrentPrice); err != nil {
		return fmt.Errorf("%s error setting current_price: %v", errorPrefix, err)
	}
	if err = d.Set("change_time", props.ChangeTime.String()); err != nil {
		return fmt.Errorf("%s error setting change_time: %v", errorPrefix, err)
	}
	if err = d.Set("create_time", props.CreateTime.String()); err != nil {
		return fmt.Errorf("%s error setting create_time: %v", errorPrefix, err)
	}
	if err = d.Set("status", props.Status); err != nil {
		return fmt.Errorf("%s error setting status: %v", errorPrefix, err)
	}

	k8sReleaseTemplateUUIDMap, err := getK8sReleaseTemplateUUIDMap(client)
	if err != nil {
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}

	// Get k8s release number based on paas_service_template_uuid
	var validTemplateUUID bool
	for k, v := range k8sReleaseTemplateUUIDMap {
		if v == props.ServiceTemplateUUID {
			validTemplateUUID = true
			if err = d.Set("k8s_release_computed", k); err != nil {
				return fmt.Errorf("%s error setting k8s_release_computed: %v", errorPrefix, err)
			}
			break
		}
	}
	if !validTemplateUUID {
		return fmt.Errorf(
			"%s error setting k8s_release_computed: could not find a release number of k8s service template UUID %s",
			errorPrefix,
			props.ServiceTemplateUUID)
	}

	//Get listen ports
	listenPorts := make([]interface{}, 0)
	for _, value := range props.ListenPorts {
		for k, portValue := range value {
			port := map[string]interface{}{
				"name": k,
				"port": portValue,
			}
			listenPorts = append(listenPorts, port)
		}
	}
	if err = d.Set("listen_port", listenPorts); err != nil {
		return fmt.Errorf("%s error setting listen ports: %v", errorPrefix, err)
	}

	//Get parameters
	if err = d.Set("worker_node_ram", props.Parameters["k8s_worker_node_ram"]); err != nil {
		return fmt.Errorf("%s error setting worker_node_ram: %v", errorPrefix, err)
	}
	if err = d.Set("worker_node_cores", props.Parameters["k8s_worker_node_cores"]); err != nil {
		return fmt.Errorf("%s error setting worker_node_cores: %v", errorPrefix, err)
	}
	if err = d.Set("worker_node_count", props.Parameters["k8s_worker_node_count"]); err != nil {
		return fmt.Errorf("%s error setting worker_node_count: %v", errorPrefix, err)
	}
	if err = d.Set("worker_node_storage", props.Parameters["k8s_worker_node_storage"]); err != nil {
		return fmt.Errorf("%s error setting worker_node_storage: %v", errorPrefix, err)
	}
	if err = d.Set("worker_node_storage_type", props.Parameters["k8s_worker_node_storage_type"]); err != nil {
		return fmt.Errorf("%s error setting worker_node_storage_type: %v", errorPrefix, err)
	}

	//Set labels
	if err = d.Set("labels", props.Labels); err != nil {
		return fmt.Errorf("%s error setting labels: %v", errorPrefix, err)
	}

	//Get all available networks
	networks, err := client.GetNetworkList(context.Background())
	if err != nil {
		return fmt.Errorf("%s error getting networks: %v", errorPrefix, err)
	}
	//look for a network that the PaaS service is in
	for _, network := range networks {
		securityZones := network.Properties.Relations.PaaSSecurityZones
		//Each network can contain only one security zone
		if len(securityZones) >= 1 {
			if securityZones[0].ObjectUUID == props.SecurityZoneUUID {
				if err = d.Set("network_uuid", network.Properties.ObjectUUID); err != nil {
					return fmt.Errorf("%s error setting network_uuid: %v", errorPrefix, err)
				}
			}
		}
	}
	return nil
}

func resourceGridscaleK8sCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gsclient.Client)
	errorPrefix := fmt.Sprintf("create k8s (%s) resource -", d.Id())

	k8sReleaseTemplateUUIDMap, err := getK8sReleaseTemplateUUIDMap(client)
	if err != nil {
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}
	// Check if the k8s release number exists
	templateUUID, ok := k8sReleaseTemplateUUIDMap[d.Get("k8s_release").(string)]
	if !ok {
		var releases []string
		for releaseNo := range k8sReleaseTemplateUUIDMap {
			releases = append(releases, releaseNo)
		}
		return fmt.Errorf("%v is not a valid kubernetes release number. Valid release numbers are: %v", d.Get("k8s_release").(string), strings.Join(releases, ","))
	}

	requestBody := gsclient.PaaSServiceCreateRequest{
		Name:                    d.Get("name").(string),
		PaaSServiceTemplateUUID: templateUUID,
		Labels:                  convSOStrings(d.Get("labels").(*schema.Set).List()),
		PaaSSecurityZoneUUID:    d.Get("security_zone_uuid").(string),
	}

	params := make(map[string]interface{})
	params["k8s_worker_node_ram"] = d.Get("worker_node_ram")
	params["k8s_worker_node_cores"] = d.Get("worker_node_cores")
	params["k8s_worker_node_count"] = d.Get("worker_node_count")
	params["k8s_worker_node_storage"] = d.Get("worker_node_storage")
	params["k8s_worker_node_storage_type"] = d.Get("worker_node_storage_type")
	requestBody.Parameters = params

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()
	response, err := client.CreatePaaSService(ctx, requestBody)
	if err != nil {
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}
	d.SetId(response.ObjectUUID)
	log.Printf("The id for PaaS service %s has been set to %v", requestBody.Name, response.ObjectUUID)
	return resourceGridscaleK8sRead(d, meta)
}

func resourceGridscaleK8sUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gsclient.Client)
	errorPrefix := fmt.Sprintf("update k8s (%s) resource -", d.Id())

	k8sReleaseTemplateUUIDMap, err := getK8sReleaseTemplateUUIDMap(client)
	if err != nil {
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}

	labels := convSOStrings(d.Get("labels").(*schema.Set).List())
	requestBody := gsclient.PaaSServiceUpdateRequest{
		Name:   d.Get("name").(string),
		Labels: &labels,
	}

	// Only update k8s_release, when it is changed
	if d.HasChange("k8s_release") {
		// Check if the k8s release number exists
		templateUUID, ok := k8sReleaseTemplateUUIDMap[d.Get("k8s_release").(string)]
		if !ok {
			var releases []string
			for releaseNo := range k8sReleaseTemplateUUIDMap {
				releases = append(releases, releaseNo)
			}
			return fmt.Errorf("%v is not a valid kubernetes release number. Valid release numbers are: %v", d.Get("k8s_release").(string), strings.Join(releases, ","))
		}
		requestBody.PaaSServiceTemplateUUID = templateUUID
	}

	params := make(map[string]interface{})
	params["k8s_worker_node_ram"] = d.Get("worker_node_ram")
	params["k8s_worker_node_cores"] = d.Get("worker_node_cores")
	params["k8s_worker_node_count"] = d.Get("worker_node_count")
	params["k8s_worker_node_storage"] = d.Get("worker_node_storage")
	params["k8s_worker_node_storage_type"] = d.Get("worker_node_storage_type")
	requestBody.Parameters = params

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()
	err = client.UpdatePaaSService(ctx, d.Id(), requestBody)
	if err != nil {
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}
	return resourceGridscaleK8sRead(d, meta)
}

func resourceGridscaleK8sDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gsclient.Client)
	errorPrefix := fmt.Sprintf("delete k8s (%s) resource -", d.Id())

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()
	err := errHandler.RemoveErrorContainsHTTPCodes(
		client.DeletePaaSService(ctx, d.Id()),
		http.StatusNotFound,
	)
	if err != nil {
		return fmt.Errorf("%s error: %v", errorPrefix, err)
	}
	return nil
}

// getK8sReleaseTemplateUUIDMap gets all k8s service templates' release numbers and their UUIDs.
// Returns a map where release numbers are keys and UUIDs are values.
func getK8sReleaseTemplateUUIDMap(client *gsclient.Client) (map[string]string, error) {
	k8sReleaseTemplateUUIDMap := make(map[string]string)
	// Get all PaaS service templates
	// for validating PaaS resource purposes
	paasTemplates, err := client.GetPaaSTemplateList(context.Background())
	if err != nil {
		return k8sReleaseTemplateUUIDMap, err
	}

	// Get k8s releases and corresponding UUIDs
	for _, template := range paasTemplates {
		if template.Properties.Category == k8sTemplateCategoryName {
			k8sReleaseTemplateUUIDMap[template.Properties.Release] = template.Properties.ObjectUUID
		}
	}
	return k8sReleaseTemplateUUIDMap, nil
}
