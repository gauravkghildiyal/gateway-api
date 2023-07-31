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
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	gatewayv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// TODO(gauravkghildiyal): Not ported because of dependent child functions called from within.
//
// ValidateHTTPRoute validates HTTPRoute according to the Gateway API specification.
// For additional details of the HTTPRoute spec, refer to:
// https://gateway-api.sigs.k8s.io/v1beta1/references/spec/#gateway.networking.k8s.io/v1beta1.HTTPRoute
func ValidateHTTPRoute(route *gatewayv1b1.HTTPRoute) field.ErrorList {
	return ValidateHTTPRouteSpec(&route.Spec, field.NewPath("spec"))
}

// TODO(gauravkghildiyal): Not ported because of dependent child functions called from within.
//
// ValidateHTTPRouteSpec validates that required fields of spec are set according to the
// HTTPRoute specification.
func ValidateHTTPRouteSpec(spec *gatewayv1b1.HTTPRouteSpec, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	for i, rule := range spec.Rules {
		errs = append(errs, validateHTTPRouteFilters(rule.Filters, rule.Matches, path.Child("rules").Index(i))...)
		for j, backendRef := range rule.BackendRefs {
			errs = append(errs, validateHTTPRouteFilters(backendRef.Filters, rule.Matches, path.Child("rules").Index(i).Child("backendRefs").Index(j))...)
		}
		for j, m := range rule.Matches {
			matchPath := path.Child("rules").Index(i).Child("matches").Index(j)

			if len(m.Headers) > 0 {
				errs = append(errs, validateHTTPHeaderMatches(m.Headers, matchPath.Child("headers"))...)
			}
		}
	}

	// TODO(gauravkghildiyal): The following conversion has two problems:
	//  1. For some reason, we're aren't able to compare the "namespace" field and observe the error:
	//    "undefined field 'namespace'". This needs to be explored further.
	//
	//  2. Comparing 'port' field is also a problem because it is only available in
	//     Experimental and not in Standard. So we need different validation rules
	//     for both of them (or another way to make thsi work)
	//
	// Applied the following annotation on the CommonRouteSpec.ParentRefs array:
	//
	// +kubebuilder:validation:XValidation:message="sectionName or port must be unique when parentRefs includes 2 or more references to the same parent",rule="self.all(p1, self.exists_one(p2, p1.group == p2.group && p1.kind == p2.kind && ((!has(p1.namespace) && !has(p2.namespace)) || (!has(p1.namespace) && p2.namespace == '') || (p1.namespace == '' && !has(p2.namespace)) || (p1.namespace == p2.namespace)) && p1.name == p2.name && ((!has(p1.sectionName) && !has(p2.sectionName)) || (!has(p1.sectionName) && p2.sectionName == '') || (p1.sectionName == '' && !has(p2.sectionName)) || (p1.sectionName == p2.sectionName)) && ((!has(p1.port) && !has(p2.port)) || (!has(p1.port) && p2.port == '') || (p1.port == '' && !has(p2.port)) || (p1.port == p2.port))))"
	errs = append(errs, ValidateParentRefs(spec.ParentRefs, path.Child("spec"))...)
	return errs
}

// TODO(gauravkghildiyal): Not ported because of dependent child functions called from within.
//
// validateHTTPRouteFilters validates that a list of core and extended filters
// is used at most once and that the filter type matches its value
func validateHTTPRouteFilters(filters []gatewayv1b1.HTTPRouteFilter, matches []gatewayv1b1.HTTPRouteMatch, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	counts := map[gatewayv1b1.HTTPRouteFilterType]int{}

	for i, filter := range filters {
		counts[filter.Type]++
		if filter.RequestHeaderModifier != nil {
			errs = append(errs, validateHTTPHeaderModifier(*filter.RequestHeaderModifier, path.Index(i).Child("requestHeaderModifier"))...)
		}
		if filter.ResponseHeaderModifier != nil {
			errs = append(errs, validateHTTPHeaderModifier(*filter.ResponseHeaderModifier, path.Index(i).Child("responseHeaderModifier"))...)
		}
	}

	if counts[gatewayv1b1.HTTPRouteFilterRequestRedirect] > 0 && counts[gatewayv1b1.HTTPRouteFilterURLRewrite] > 0 {
		errs = append(errs, field.Invalid(path.Child("filters"), gatewayv1b1.HTTPRouteFilterRequestRedirect, "may specify either httpRouteFilterRequestRedirect or httpRouteFilterRequestRewrite, but not both"))
	}
	return errs
}

// TODO(gauravkghildiyal): Cost exceeded.
//
// +kubebuilder:validation:XValidation:message="Must not match the same header (case-insensitive) multiple times in the same rule",rule="self.all(h1, self.exists_one(h2, h1.name.lowerAscii() == h2.name.lowerAscii()))"
//
// validateHTTPHeaderMatches validates that no header name
// is matched more than once (case-insensitive).
func validateHTTPHeaderMatches(matches []gatewayv1b1.HTTPHeaderMatch, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	counts := map[string]int{}

	for _, match := range matches {
		// Header names are case-insensitive.
		counts[strings.ToLower(string(match.Name))]++
	}

	for name, count := range counts {
		if count > 1 {
			errs = append(errs, field.Invalid(path, http.CanonicalHeaderKey(name), "cannot match the same header multiple times in the same rule"))
		}
	}

	return errs
}

// TODO(gauravkghildiyal): Cost exceeded.
//
// +kubebuilder:validation:XValidation:message="Must not intersect",rule="self.set.all(e1, self.add.exists_one(e2, e1.name.lowerAscii() == e2.name.lowerAscii()))"
func validateHTTPHeaderModifier(filter gatewayv1b1.HTTPHeaderFilter, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	singleAction := make(map[string]bool)
	for i, action := range filter.Add {
		if needsErr, ok := singleAction[strings.ToLower(string(action.Name))]; ok {
			if needsErr {
				errs = append(errs, field.Invalid(path.Child("add"), filter.Add[i], "cannot specify multiple actions for header"))
			}
			singleAction[strings.ToLower(string(action.Name))] = false
		} else {
			singleAction[strings.ToLower(string(action.Name))] = true
		}
	}
	for i, action := range filter.Set {
		if needsErr, ok := singleAction[strings.ToLower(string(action.Name))]; ok {
			if needsErr {
				errs = append(errs, field.Invalid(path.Child("set"), filter.Set[i], "cannot specify multiple actions for header"))
			}
			singleAction[strings.ToLower(string(action.Name))] = false
		} else {
			singleAction[strings.ToLower(string(action.Name))] = true
		}
	}
	for i, name := range filter.Remove {
		if needsErr, ok := singleAction[strings.ToLower(name)]; ok {
			if needsErr {
				errs = append(errs, field.Invalid(path.Child("remove"), filter.Remove[i], "cannot specify multiple actions for header"))
			}
			singleAction[strings.ToLower(name)] = false
		} else {
			singleAction[strings.ToLower(name)] = true
		}
	}
	return errs
}
