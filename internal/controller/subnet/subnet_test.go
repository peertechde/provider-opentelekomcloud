package subnet

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opentelekomcloud/gophertelekomcloud/testhelper"
	fake "github.com/opentelekomcloud/gophertelekomcloud/testhelper/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/subnet/v1alpha1"
	"github.com/peertechde/provider-opentelekomcloud/internal/pointer"
)

type params func(*v1alpha1.Subnet)

func subnet(p ...params) *v1alpha1.Subnet {
	s := &v1alpha1.Subnet{}
	for _, f := range p {
		f(s)
	}
	return s
}

func withExternalName(name string) params {
	return func(s *v1alpha1.Subnet) {
		meta.SetExternalName(s, name)
	}
}

func withSpec(name, cidr, gatewayIP, vpcID string) params {
	return func(s *v1alpha1.Subnet) {
		s.Spec.ForProvider.Name = name
		s.Spec.ForProvider.CIDR = cidr
		s.Spec.ForProvider.GatewayIP = gatewayIP
		s.Spec.ForProvider.VPCID = vpcID
	}
}

func withDHCPOption(enable bool) params {
	return func(s *v1alpha1.Subnet) {
		s.Spec.ForProvider.DHCPEnable = pointer.To(enable)
	}
}

func TestObserve(t *testing.T) {
	type fields struct {
		handler http.HandlerFunc
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"SubnetNotFound": {
			reason: "Should return ResourceExists: false when API returns 404",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					testhelper.TestMethod(t, r, "GET")
					w.WriteHeader(http.StatusNotFound)
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  subnet(withExternalName("subnet-id-123")),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
			},
		},
		"APIError": {
			reason: "Should return an error when API fails unexpectedly",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  subnet(withExternalName("subnet-id-123")),
			},
			want: want{
				err: fmt.Errorf("cannot observe Subnet"),
			},
		},
		"UpToDate": {
			reason: "Should observe resource as existing and up to date",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					testhelper.TestMethod(t, r, "GET")
					w.Header().Add("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)

					fmt.Fprintf(w, `
						{
							"subnet": {
								"id": "subnet-id-123",
								"name": "test-subnet",
								"cidr": "192.168.1.0/24",
								"gateway_ip": "192.168.1.1",
								"vpc_id": "vpc-123",
								"status": "ACTIVE",
								"dhcp_enable": true
							}
						}
					`)
				},
			},
			args: args{
				ctx: context.Background(),
				mg: subnet(
					withExternalName("subnet-id-123"),
					withSpec("test-subnet", "192.168.1.0/24", "192.168.1.1", "vpc-123"),
					withDHCPOption(true),
				),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		"DriftDetected": {
			reason: "Should detect drift when remote state (e.g., Name) differs from spec",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					testhelper.TestMethod(t, r, "GET")
					w.Header().Add("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)

					// Remote has "old-name", Spec has "new-name"
					fmt.Fprintf(w, `
						{
							"subnet": {
								"id": "subnet-id-123",
								"name": "old-name",
								"cidr": "192.168.1.0/24",
								"gateway_ip": "192.168.1.1",
								"vpc_id": "vpc-123",
								"status": "ACTIVE"
							}
						}
					`)
				},
			},
			args: args{
				ctx: context.Background(),
				mg: subnet(
					withExternalName("subnet-id-123"),
					withSpec("new-name", "192.168.1.0/24", "192.168.1.1", "vpc-123"),
					withDHCPOption(false),
				),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        false,
					ResourceLateInitialized: false,
				},
			},
		},
		"LateInitialization": {
			reason: "Should populate optional fields (Late Init) if missing in Spec but present in Cloud",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					testhelper.TestMethod(t, r, "GET")
					w.Header().Add("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)

					// Provider has dhcp_enable: true, Spec has nil
					fmt.Fprintf(w, `
						{
							"subnet": {
								"id": "subnet-id-123",
								"name": "test-subnet",
								"cidr": "192.168.1.0/24",
								"gateway_ip": "192.168.1.1",
								"vpc_id": "vpc-123",
								"status": "ACTIVE",
								"dhcp_enable": true
							}
						}
					`)
				},
			},
			args: args{
				ctx: context.Background(),
				mg: subnet(
					withExternalName("subnet-id-123"),
					withSpec("test-subnet", "192.168.1.0/24", "192.168.1.1", "vpc-123"),
					// DHCP is nil here
				),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceUpToDate:        true,
					ResourceLateInitialized: true,
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup a mock server
			testhelper.SetupHTTP()
			defer testhelper.TeardownHTTP()

			// Configure the mux to handle the specific path for this test case
			if tc.fields.handler != nil {
				// The SDK uses /subnets/{id}
				extName := meta.GetExternalName(tc.args.mg)
				testhelper.Mux.HandleFunc("/subnets/"+extName, tc.fields.handler)
			}

			// Create a fake client pointing to the mock server
			sc := fake.ServiceClient()
			sc.Endpoint = testhelper.Endpoint()

			e := external{client: sc}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			if tc.want.err != nil {
				if err == nil {
					t.Errorf("\n%s\ne.Observe(...): -want error, +got nil\n", tc.reason)
				} else if !strings.Contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("\n%s\ne.Observe(...): -want error containing %q, +got %q\n", tc.reason, tc.want.err.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("\n%s\ne.Observe(...): -want nil, +got error %v\n", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
