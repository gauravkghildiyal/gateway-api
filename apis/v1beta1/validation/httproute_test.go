/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	gatewayv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestValidateHTTPRoute(t *testing.T) {
	tests := []struct {
		name     string
		rules    []gatewayv1b1.HTTPRouteRule
		errCount int
	}{{
		name:     "redirect path modifier with type mismatch",
		errCount: 2,
		rules: []gatewayv1b1.HTTPRouteRule{{
			Filters: []gatewayv1b1.HTTPRouteFilter{{
				Type: gatewayv1b1.HTTPRouteFilterRequestRedirect,
				RequestRedirect: &gatewayv1b1.HTTPRequestRedirectFilter{
					Path: &gatewayv1b1.HTTPPathModifier{
						Type:            gatewayv1b1.PrefixMatchHTTPPathModifier,
						ReplaceFullPath: ptrTo("foo"),
					},
				},
			}},
		}},
	}, {
		name:     "rewrite path modifier missing path match",
		errCount: 1,
		rules: []gatewayv1b1.HTTPRouteRule{{
			Filters: []gatewayv1b1.HTTPRouteFilter{{
				Type: gatewayv1b1.HTTPRouteFilterURLRewrite,
				URLRewrite: &gatewayv1b1.HTTPURLRewriteFilter{
					Path: &gatewayv1b1.HTTPPathModifier{
						Type:               gatewayv1b1.PrefixMatchHTTPPathModifier,
						ReplacePrefixMatch: ptrTo("foo"),
					},
				},
			}},
		}},
	}, {
		name:     "multiple actions for the same request header (invalid)",
		errCount: 2,
		rules: []gatewayv1b1.HTTPRouteRule{{
			Filters: []gatewayv1b1.HTTPRouteFilter{{
				Type: gatewayv1b1.HTTPRouteFilterRequestHeaderModifier,
				RequestHeaderModifier: &gatewayv1b1.HTTPHeaderFilter{
					Add: []gatewayv1b1.HTTPHeader{
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-fruit"),
							Value: "apple",
						},
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-vegetable"),
							Value: "carrot",
						},
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-grain"),
							Value: "rye",
						},
					},
					Set: []gatewayv1b1.HTTPHeader{
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-fruit"),
							Value: "watermelon",
						},
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-grain"),
							Value: "wheat",
						},
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-spice"),
							Value: "coriander",
						},
					},
				},
			}},
		}},
	}, {
		name:     "multiple actions for the same request header with inconsistent case (invalid)",
		errCount: 1,
		rules: []gatewayv1b1.HTTPRouteRule{{
			Filters: []gatewayv1b1.HTTPRouteFilter{{
				Type: gatewayv1b1.HTTPRouteFilterRequestHeaderModifier,
				RequestHeaderModifier: &gatewayv1b1.HTTPHeaderFilter{
					Add: []gatewayv1b1.HTTPHeader{
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-fruit"),
							Value: "apple",
						},
					},
					Set: []gatewayv1b1.HTTPHeader{
						{
							Name:  gatewayv1b1.HTTPHeaderName("X-Fruit"),
							Value: "watermelon",
						},
					},
				},
			}},
		}},
	}, {
		// TODO(gauravkghildiyal): The following validation may already by covered
		// by OpenAPIv3 validation so we may not require a CEL validation for this.
		name:     "multiple of the same action for the same request header (invalid)",
		errCount: 1,
		rules: []gatewayv1b1.HTTPRouteRule{{
			Filters: []gatewayv1b1.HTTPRouteFilter{{
				Type: gatewayv1b1.HTTPRouteFilterRequestHeaderModifier,
				RequestHeaderModifier: &gatewayv1b1.HTTPHeaderFilter{
					Add: []gatewayv1b1.HTTPHeader{
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-fruit"),
							Value: "apple",
						},
						{
							Name:  gatewayv1b1.HTTPHeaderName("x-fruit"),
							Value: "plum",
						},
					},
				},
			}},
		}},
	}, {
		name:     "multiple actions for the same response header (invalid)",
		errCount: 1,
		rules: []gatewayv1b1.HTTPRouteRule{{
			Filters: []gatewayv1b1.HTTPRouteFilter{{
				Type: gatewayv1b1.HTTPRouteFilterResponseHeaderModifier,
				ResponseHeaderModifier: &gatewayv1b1.HTTPHeaderFilter{
					Add: []gatewayv1b1.HTTPHeader{{
						Name:  gatewayv1b1.HTTPHeaderName("x-example"),
						Value: "blueberry",
					}},
					Set: []gatewayv1b1.HTTPHeader{{
						Name:  gatewayv1b1.HTTPHeaderName("x-example"),
						Value: "turnip",
					}},
				},
			}},
		}},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var errs field.ErrorList
			route := gatewayv1b1.HTTPRoute{Spec: gatewayv1b1.HTTPRouteSpec{Rules: tc.rules}}
			errs = ValidateHTTPRoute(&route)
			if len(errs) != tc.errCount {
				t.Errorf("got %d errors, want %d errors: %s", len(errs), tc.errCount, errs)
			}
		})
	}
}

func TestValidateHTTPBackendUniqueFilters(t *testing.T) {
	var testService gatewayv1b1.ObjectName = "testService"
	var specialService gatewayv1b1.ObjectName = "specialService"
	tests := []struct {
		name     string
		rules    []gatewayv1b1.HTTPRouteRule
		errCount int
	}{{
		name:     "valid httpRoute Rules backendref filters",
		errCount: 0,
		rules: []gatewayv1b1.HTTPRouteRule{{
			BackendRefs: []gatewayv1b1.HTTPBackendRef{
				{
					BackendRef: gatewayv1b1.BackendRef{
						BackendObjectReference: gatewayv1b1.BackendObjectReference{
							Name: testService,
							Port: ptrTo(gatewayv1b1.PortNumber(8080)),
						},
						Weight: ptrTo(int32(100)),
					},
					Filters: []gatewayv1b1.HTTPRouteFilter{
						{
							Type: gatewayv1b1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayv1b1.HTTPRequestMirrorFilter{
								BackendRef: gatewayv1b1.BackendObjectReference{
									Name: testService,
									Port: ptrTo(gatewayv1b1.PortNumber(8080)),
								},
							},
						},
					},
				},
			},
		}},
	}, {
		name:     "valid httpRoute Rules duplicate mirror filter",
		errCount: 0,
		rules: []gatewayv1b1.HTTPRouteRule{{
			BackendRefs: []gatewayv1b1.HTTPBackendRef{
				{
					BackendRef: gatewayv1b1.BackendRef{
						BackendObjectReference: gatewayv1b1.BackendObjectReference{
							Name: testService,
							Port: ptrTo(gatewayv1b1.PortNumber(8080)),
						},
					},
					Filters: []gatewayv1b1.HTTPRouteFilter{
						{
							Type: gatewayv1b1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayv1b1.HTTPRequestMirrorFilter{
								BackendRef: gatewayv1b1.BackendObjectReference{
									Name: testService,
									Port: ptrTo(gatewayv1b1.PortNumber(8080)),
								},
							},
						},
						{
							Type: gatewayv1b1.HTTPRouteFilterRequestMirror,
							RequestMirror: &gatewayv1b1.HTTPRequestMirrorFilter{
								BackendRef: gatewayv1b1.BackendObjectReference{
									Name: specialService,
									Port: ptrTo(gatewayv1b1.PortNumber(8080)),
								},
							},
						},
					},
				},
			},
		}},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			route := gatewayv1b1.HTTPRoute{Spec: gatewayv1b1.HTTPRouteSpec{Rules: tc.rules}}
			errs := ValidateHTTPRoute(&route)
			if len(errs) != tc.errCount {
				t.Errorf("got %d errors, want %d errors: %s", len(errs), tc.errCount, errs)
			}
		})
	}
}

func TestValidateHTTPHeaderMatches(t *testing.T) {
	tests := []struct {
		name          string
		headerMatches []gatewayv1b1.HTTPHeaderMatch
		expectErr     string
	}{{
		name:          "no header matches",
		headerMatches: nil,
		expectErr:     "",
	}, {
		name: "no header matched more than once",
		headerMatches: []gatewayv1b1.HTTPHeaderMatch{
			{Name: "Header-Name-1", Value: "val-1"},
			{Name: "Header-Name-2", Value: "val-2"},
			{Name: "Header-Name-3", Value: "val-3"},
		},
		expectErr: "",
	}, {
		name: "header matched more than once (same case)",
		headerMatches: []gatewayv1b1.HTTPHeaderMatch{
			{Name: "Header-Name-1", Value: "val-1"},
			{Name: "Header-Name-2", Value: "val-2"},
			{Name: "Header-Name-1", Value: "val-3"},
		},
		expectErr: "spec.rules[0].matches[0].headers: Invalid value: \"Header-Name-1\": cannot match the same header multiple times in the same rule",
	}, {
		name: "header matched more than once (different case)",
		headerMatches: []gatewayv1b1.HTTPHeaderMatch{
			{Name: "Header-Name-1", Value: "val-1"},
			{Name: "Header-Name-2", Value: "val-2"},
			{Name: "HEADER-NAME-2", Value: "val-3"},
		},
		expectErr: "spec.rules[0].matches[0].headers: Invalid value: \"Header-Name-2\": cannot match the same header multiple times in the same rule",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			route := gatewayv1b1.HTTPRoute{Spec: gatewayv1b1.HTTPRouteSpec{
				Rules: []gatewayv1b1.HTTPRouteRule{{
					Matches: []gatewayv1b1.HTTPRouteMatch{{
						Headers: tc.headerMatches,
					}},
					BackendRefs: []gatewayv1b1.HTTPBackendRef{{
						BackendRef: gatewayv1b1.BackendRef{
							BackendObjectReference: gatewayv1b1.BackendObjectReference{
								Name: gatewayv1b1.ObjectName("test"),
								Port: ptrTo(gatewayv1b1.PortNumber(8080)),
							},
						},
					}},
				}},
			}}

			errs := ValidateHTTPRoute(&route)
			if len(tc.expectErr) == 0 {
				assert.Emptyf(t, errs, "expected no errors, got %d errors: %s", len(errs), errs)
			} else {
				require.Lenf(t, errs, 1, "expected one error, got %d errors: %s", len(errs), errs)
				assert.Equal(t, tc.expectErr, errs[0].Error())
			}
		})
	}
}

func TestValidateHTTPQueryParamMatches(t *testing.T) {
	tests := []struct {
		name              string
		queryParamMatches []gatewayv1b1.HTTPQueryParamMatch
		expectErr         string
	}{{
		name:              "no query param matches",
		queryParamMatches: nil,
		expectErr:         "",
	}, {
		name: "no query param matched more than once",
		queryParamMatches: []gatewayv1b1.HTTPQueryParamMatch{
			{Name: "query-param-1", Value: "val-1"},
			{Name: "query-param-2", Value: "val-2"},
			{Name: "query-param-3", Value: "val-3"},
		},
		expectErr: "",
	}, {
		name: "query param matched more than once",
		queryParamMatches: []gatewayv1b1.HTTPQueryParamMatch{
			{Name: "query-param-1", Value: "val-1"},
			{Name: "query-param-2", Value: "val-2"},
			{Name: "query-param-1", Value: "val-3"},
		},
		expectErr: "spec.rules[0].matches[0].queryParams: Invalid value: \"query-param-1\": cannot match the same query parameter multiple times in the same rule",
	}, {
		name: "query param names with different casing are not considered duplicates",
		queryParamMatches: []gatewayv1b1.HTTPQueryParamMatch{
			{Name: "query-param-1", Value: "val-1"},
			{Name: "query-param-2", Value: "val-2"},
			{Name: "QUERY-PARAM-1", Value: "val-3"},
		},
		expectErr: "",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			route := gatewayv1b1.HTTPRoute{Spec: gatewayv1b1.HTTPRouteSpec{
				Rules: []gatewayv1b1.HTTPRouteRule{{
					Matches: []gatewayv1b1.HTTPRouteMatch{{
						QueryParams: tc.queryParamMatches,
					}},
					BackendRefs: []gatewayv1b1.HTTPBackendRef{{
						BackendRef: gatewayv1b1.BackendRef{
							BackendObjectReference: gatewayv1b1.BackendObjectReference{
								Name: gatewayv1b1.ObjectName("test"),
								Port: ptrTo(gatewayv1b1.PortNumber(8080)),
							},
						},
					}},
				}},
			}}

			errs := ValidateHTTPRoute(&route)
			if len(tc.expectErr) == 0 {
				assert.Emptyf(t, errs, "expected no errors, got %d errors: %s", len(errs), errs)
			} else {
				require.Lenf(t, errs, 1, "expected one error, got %d errors: %s", len(errs), errs)
				assert.Equal(t, tc.expectErr, errs[0].Error())
			}
		})
	}
}
