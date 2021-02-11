//
// Copyright 2021 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package service

import (
	"time"

	operatorv1alpha1 "github.com/ibm/ibm-licensing-operator/api/v1alpha1"
	rhmpcommon "github.com/redhat-marketplace/redhat-marketplace-operator/v2/apis/marketplace/common"
	rhmp "github.com/redhat-marketplace/redhat-marketplace-operator/v2/apis/marketplace/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetMeterDefinition(instance *operatorv1alpha1.IBMLicensing) []*rhmp.MeterDefinition {
	return []*rhmp.MeterDefinition{getCloudPakMeterDefinition(instance), getProductMeterDefinition(instance)}
}

func getCloudPakMeterDefinition(instance *operatorv1alpha1.IBMLicensing) *rhmp.MeterDefinition {
	return &rhmp.MeterDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetMeterDefinitionName(instance, true),
			Namespace: instance.Spec.InstanceNamespace,
		},
		Spec: rhmp.MeterDefinitionSpec{
			Group: "{{ .Label.productId}}.licensing.ibm.com",
			Kind:  "IBMLicensing",
			ResourceFilters: []rhmp.ResourceFilter{
				{
					Namespace: &rhmp.NamespaceFilter{
						UseOperatorGroup: true,
					},
					OwnerCRD: &rhmp.OwnerCRDFilter{
						GroupVersionKind: rhmpcommon.GroupVersionKind{
							APIVersion: "operator.ibm.com/v1alpha1",
							Kind:       "IBMLicensing",
						},
					},
					WorkloadType: rhmp.WorkloadTypeService,
				},
			},
			Meters: []rhmp.MeterWorkload{
				{
					Name:               "{{ .Label.productId}}.licensing.ibm.com",
					Aggregation:        "max",
					Period:             &metav1.Duration{Duration: 24 * time.Hour},
					WorkloadType:       rhmp.WorkloadTypeService,
					Metric:             "{{ .Label.metricId}}",
					Query:              "product_license_usage{}",
					GroupBy:            []string{"metricId", "productId"},
					ValueLabelOverride: "{{ .Label.value}}",
					DateLabelOverride:  "{{ .Label.date}}",
				},
			},
		},
	}
}

func getProductMeterDefinition(instance *operatorv1alpha1.IBMLicensing) *rhmp.MeterDefinition {
	return &rhmp.MeterDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetMeterDefinitionName(instance, false),
			Namespace: instance.Spec.InstanceNamespace,
		},
		Spec: rhmp.MeterDefinitionSpec{
			Group: "{{ .Label.productId}}.licensing.ibm.com",
			Kind:  "IBMLicensing",
			ResourceFilters: []rhmp.ResourceFilter{
				{
					Namespace: &rhmp.NamespaceFilter{
						UseOperatorGroup: true,
					},
					OwnerCRD: &rhmp.OwnerCRDFilter{
						GroupVersionKind: rhmpcommon.GroupVersionKind{
							APIVersion: "operator.ibm.com/v1alpha1",
							Kind:       "IBMLicensing",
						},
					},
					WorkloadType: rhmp.WorkloadTypeService,
				},
			},
			Meters: []rhmp.MeterWorkload{
				{
					Name:               "{{ .Label.productId}}.licensing.ibm.com",
					Aggregation:        "max",
					Period:             &metav1.Duration{Duration: 24 * time.Hour},
					WorkloadType:       rhmp.WorkloadTypeService,
					Metric:             "{{ .Label.metricId}}",
					Query:              "product_license_usage_details{}",
					GroupBy:            []string{"metricId", "productId"},
					ValueLabelOverride: "{{ .Label.value}}",
					DateLabelOverride:  "{{ .Label.date}}",
				},
			},
		},
	}
}

func GetMeterDefinitionName(instance *operatorv1alpha1.IBMLicensing, isCloudpak bool) string {
	if isCloudpak {
		return LicensingResourceBase + "-product-" + instance.GetName()
	}
	return LicensingResourceBase + "-bundleproduct-" + instance.GetName()

}