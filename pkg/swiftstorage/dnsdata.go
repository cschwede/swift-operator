package swiftstorage

import (
	"context"
	"fmt"

	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// DNSData - Create DNS entry that openstack dnsmasq will resolve
func DNSData(
	ctx context.Context,
	helper *helper.Helper,
	hostName string,
	ip string,
	swiftPod corev1.Pod,
	serviceLabels map[string]string,
) error {
	dnsHostCname := infranetworkv1.DNSHost{
		IP: ip,
		Hostnames: []string{
			hostName,
		},
	}

	// Create DNSData object
	dnsData := &infranetworkv1.DNSData{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dns-" + swiftPod.Name,
			Namespace: swiftPod.Namespace,
			Labels:    serviceLabels,
		},
	}
	dnsHosts := []infranetworkv1.DNSHost{dnsHostCname}

	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), dnsData, func() error {
		dnsData.Spec.Hosts = dnsHosts
		// TODO: use value from DNSMasq instance instead of hardcode
		dnsData.Spec.DNSDataLabelSelectorValue = "dnsdata"
		err := controllerutil.SetControllerReference(helper.GetBeforeObject(), dnsData, helper.GetScheme())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error creating DNSData %s: %w", dnsData.Name, err)
	}
	return nil
}
