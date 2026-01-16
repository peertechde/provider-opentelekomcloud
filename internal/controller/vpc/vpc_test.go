package vpc

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

	v1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/vpc/v1alpha1"
)

type params func(*v1alpha1.VPC)

func vpc(p ...params) *v1alpha1.VPC {
	v := &v1alpha1.VPC{}
	for _, f := range p {
		f(v)
	}
	return v
}

func withExternalName(name string) params {
	return func(v *v1alpha1.VPC) {
		meta.SetExternalName(v, name)
	}
}

func withSpec(name, cidr string) params {
	return func(v *v1alpha1.VPC) {
		v.Spec.ForProvider.Name = name
		v.Spec.ForProvider.CIDR = cidr
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
		"VPCNotFound": {
			reason: "Should return ResourceExists: false when API returns 404",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					// Mock the 404 response from OTC
					testhelper.TestMethod(t, r, "GET")
					testhelper.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
					w.WriteHeader(http.StatusNotFound)
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  vpc(withExternalName("vpc-id-123")),
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
				mg:  vpc(withExternalName("vpc-id-123")),
			},
			want: want{
				err: fmt.Errorf("cannot observe VPC"),
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
							"vpc": {
								"id": "vpc-id-123",
								"name": "test-vpc",
								"cidr": "192.168.0.0/16",
								"status": "OK"
							}
						}
					`)
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  vpc(withExternalName("vpc-id-123"), withSpec("test-vpc", "192.168.0.0/16")),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		"DriftDetected": {
			reason: "Should detect drift when remote state differs from spec",
			fields: fields{
				handler: func(w http.ResponseWriter, r *http.Request) {
					testhelper.TestMethod(t, r, "GET")
					w.Header().Add("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)

					// Remote has "old-name", Spec has "new-name"
					fmt.Fprintf(w, `
						{
							"vpc": {
								"id": "vpc-id-123",
								"name": "old-name",
								"cidr": "192.168.0.0/16",
								"status": "OK"
							}
						}
					`)
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  vpc(withExternalName("vpc-id-123"), withSpec("new-name", "192.168.0.0/16")),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false, // Drift
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
				// The SDK uses /vpcs/{id}
				extName := meta.GetExternalName(tc.args.mg)
				testhelper.Mux.HandleFunc("/vpcs/"+extName, tc.fields.handler)
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
