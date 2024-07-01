/*
Copyright The HTNN Authors.

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
package v1

func ConvertHTTPFilterPolicyToFilterPolicy(hp *HTTPFilterPolicy) FilterPolicy {
	p := FilterPolicy{
		convertedFromHTTPFilterPolicy: true,
		TypeMeta:                      hp.TypeMeta,
		ObjectMeta:                    hp.ObjectMeta,
		Spec: FilterPolicySpec{
			TargetRef:   hp.Spec.TargetRef,
			Filters:     hp.Spec.Filters,
			SubPolicies: make([]FilterSubPolicy, 0, len(hp.Spec.SubPolicies)),
		},
		// Fields in Spec is read-only, and fields in Status is not
		Status: FilterPolicyStatus{
			Conditions: hp.Status.DeepCopy().Conditions,
		},
	}
	for _, sp := range hp.Spec.SubPolicies {
		p.Spec.SubPolicies = append(p.Spec.SubPolicies, FilterSubPolicy(sp))
	}

	return p
}

func ConvertFilterPolicyToHTTPFilterPolicy(p *FilterPolicy) HTTPFilterPolicy {
	hp := HTTPFilterPolicy{
		TypeMeta:   p.TypeMeta,
		ObjectMeta: p.ObjectMeta,
		Spec: HTTPFilterPolicySpec{
			TargetRef:   p.Spec.TargetRef,
			Filters:     p.Spec.Filters,
			SubPolicies: make([]HTTPFilterSubPolicy, 0, len(p.Spec.SubPolicies)),
		},
		Status: HTTPFilterPolicyStatus{
			Conditions: p.Status.DeepCopy().Conditions,
		},
	}
	for _, sp := range p.Spec.SubPolicies {
		hp.Spec.SubPolicies = append(hp.Spec.SubPolicies, HTTPFilterSubPolicy(sp))
	}

	return hp
}
