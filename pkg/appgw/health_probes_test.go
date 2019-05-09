// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestHealthProbes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up App Gateway health probes")
}

var _ = Describe("configure App Gateway health probes", func() {

	port1 := v1.ServicePort{
		// The name of this port within the service. This must be a DNS_LABEL.
		// All ports within a ServiceSpec must have unique names. This maps to
		// the 'Name' field in EndpointPort objects.
		// Optional if only one ServicePort is defined on this service.
		Name: "http",

		// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
		Protocol: v1.ProtocolTCP,

		// The port that will be exposed by this service.
		Port: 80,

		// Number or name of the port to access on the pods targeted by the service.
		// Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
		// If this is a string, it will be looked up as a named port in the
		// target Pod's container ports. If this is not specified, the value
		// of the 'port' field is used (an identity map).
		// This field is ignored for services with clusterIP=None, and should be
		// omitted or set equal to the 'port' field.
		TargetPort: intstr.IntOrString{
			IntVal: 8181,
		},
	}

	port2 := v1.ServicePort{
		Name:     "https",
		Protocol: v1.ProtocolTCP,
		Port:     443,
		TargetPort: intstr.IntOrString{
			StrVal: "https-port",
		},
	}

	port3 := v1.ServicePort{
		Name:     "other-tcp-port",
		Protocol: v1.ProtocolTCP,
		Port:     554,
		TargetPort: intstr.IntOrString{
			IntVal: 9554,
		},
	}

	port4 := v1.ServicePort{
		Name:     "other-tcp-port",
		Protocol: v1.ProtocolUDP,
		Port:     123,
		TargetPort: intstr.IntOrString{
			IntVal: 123,
		},
	}

	Context("looking at TLS specs", func() {
		cb := makeConfigBuilderTestFixture(nil)
		endpoints := v1.Endpoints{
			Subsets: []v1.EndpointSubset{
				{
					// IP addresses which offer the related ports that are marked as ready. These endpoints
					// should be considered safe for load balancers and clients to utilize.
					// +optional
					Addresses: []v1.EndpointAddress{
						{
							IP: "10.9.8.7",
							// The Hostname of this endpoint
							// +optional
							Hostname: "www.contoso.com",
							// Optional: Node hosting this endpoint. This can be used to determine endpoints local to a node.
							// +optional
							NodeName: to.StringPtr("--node-name--"),
						},
					},
					// IP addresses which offer the related ports but are not currently marked as ready
					// because they have not yet finished starting, have recently failed a readiness check,
					// or have recently failed a liveness check.
					// +optional
					NotReadyAddresses: []v1.EndpointAddress{},
					// Port numbers available on the related IP addresses.
					// +optional
					Ports: []v1.EndpointPort{},
				},
			},
		}
		err := cb.k8sContext.Caches.Endpoints.Add(endpoints)
		It("added endpoints to cache without an error", func() {
			Expect(err).To(BeNil())
		})

		service := v1.Service{
			Spec: v1.ServiceSpec{
				// List of ports exposed by this service
				Ports: []v1.ServicePort{
					port1,
					port2,
					port3,
					port4,
				},
			},
		}
		err = cb.k8sContext.Caches.Service.Add(service)
		It("added service to cache without an error", func() {
			Expect(err).To(BeNil())
		})

		ingress := makeIngressTestFixture()
		ingressList := []*v1beta1.Ingress{
			&ingress,
		}

		// !! Action !!
		_, _ = cb.HealthProbesCollection(ingressList)
		actual := cb.appGwConfig.Probes

		expected := []network.ApplicationGatewayProbe{
			{
				ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
					Protocol:                            "Http",
					Host:                                to.StringPtr("localhost"),
					Path:                                to.StringPtr("/"),
					Interval:                            to.Int32Ptr(30),
					Timeout:                             to.Int32Ptr(30),
					UnhealthyThreshold:                  to.Int32Ptr(3),
					PickHostNameFromBackendHTTPSettings: nil,
					MinServers:                          nil,
					Match:                               nil,
					ProvisioningState:                   nil,
				},
				Name: to.StringPtr("k8s-ag-ingress-defaultprobe"),
				Etag: nil,
				Type: nil,
				ID:   nil,
			},
			{
				ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
					Protocol:                            "Http",
					Host:                                to.StringPtr(testFixturesHost),
					Path:                                to.StringPtr("/a/b/c/d/e"),
					Interval:                            to.Int32Ptr(30),
					Timeout:                             to.Int32Ptr(30),
					UnhealthyThreshold:                  to.Int32Ptr(3),
					PickHostNameFromBackendHTTPSettings: nil,
					MinServers:                          nil,
					Match:                               nil,
					ProvisioningState:                   nil,
				},
				Name: to.StringPtr("k8s-ag-ingress--8080-pb---name--"),
				Etag: nil,
				Type: nil,
				ID:   nil,
			},
			{
				ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
					Protocol:                            "Http",
					Host:                                to.StringPtr(testFixturesOtherHost),
					Path:                                to.StringPtr("/a/b/c/d/e"),
					Interval:                            to.Int32Ptr(30),
					Timeout:                             to.Int32Ptr(30),
					UnhealthyThreshold:                  to.Int32Ptr(3),
					PickHostNameFromBackendHTTPSettings: nil,
					MinServers:                          nil,
					Match:                               nil,
					ProvisioningState:                   nil,
				},
				Name: to.StringPtr("k8s-ag-ingress--8989-pb---name--"),
				Etag: nil,
				Type: nil,
				ID:   nil,
			},
		}

		It("should have exactly 3 records", func() {
			Expect(len(*actual)).To(Equal(3))
		})

		It("should succeed", func() {
			// Ensure capacities of the slices match
			Expect(*actual).To(ContainElement(expected[0]))
			Expect(*actual).To(ContainElement(expected[1]))
			Expect(*actual).To(ContainElement(expected[2]))
		})
	})
})
