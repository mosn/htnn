//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Consumer) DeepCopyInto(out *Consumer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Consumer.
func (in *Consumer) DeepCopy() *Consumer {
	if in == nil {
		return nil
	}
	out := new(Consumer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Consumer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsumerList) DeepCopyInto(out *ConsumerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Consumer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsumerList.
func (in *ConsumerList) DeepCopy() *ConsumerList {
	if in == nil {
		return nil
	}
	out := new(ConsumerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ConsumerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsumerPlugin) DeepCopyInto(out *ConsumerPlugin) {
	*out = *in
	in.Config.DeepCopyInto(&out.Config)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsumerPlugin.
func (in *ConsumerPlugin) DeepCopy() *ConsumerPlugin {
	if in == nil {
		return nil
	}
	out := new(ConsumerPlugin)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsumerSpec) DeepCopyInto(out *ConsumerSpec) {
	*out = *in
	if in.Auth != nil {
		in, out := &in.Auth, &out.Auth
		*out = make(map[string]ConsumerPlugin, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Filters != nil {
		in, out := &in.Filters, &out.Filters
		*out = make(map[string]HTTPPlugin, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsumerSpec.
func (in *ConsumerSpec) DeepCopy() *ConsumerSpec {
	if in == nil {
		return nil
	}
	out := new(ConsumerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsumerStatus) DeepCopyInto(out *ConsumerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsumerStatus.
func (in *ConsumerStatus) DeepCopy() *ConsumerStatus {
	if in == nil {
		return nil
	}
	out := new(ConsumerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPFilterPolicy) DeepCopyInto(out *HTTPFilterPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPFilterPolicy.
func (in *HTTPFilterPolicy) DeepCopy() *HTTPFilterPolicy {
	if in == nil {
		return nil
	}
	out := new(HTTPFilterPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HTTPFilterPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPFilterPolicyList) DeepCopyInto(out *HTTPFilterPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HTTPFilterPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPFilterPolicyList.
func (in *HTTPFilterPolicyList) DeepCopy() *HTTPFilterPolicyList {
	if in == nil {
		return nil
	}
	out := new(HTTPFilterPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HTTPFilterPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPFilterPolicySpec) DeepCopyInto(out *HTTPFilterPolicySpec) {
	*out = *in
	in.TargetRef.DeepCopyInto(&out.TargetRef)
	if in.Filters != nil {
		in, out := &in.Filters, &out.Filters
		*out = make(map[string]HTTPPlugin, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPFilterPolicySpec.
func (in *HTTPFilterPolicySpec) DeepCopy() *HTTPFilterPolicySpec {
	if in == nil {
		return nil
	}
	out := new(HTTPFilterPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPFilterPolicyStatus) DeepCopyInto(out *HTTPFilterPolicyStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPFilterPolicyStatus.
func (in *HTTPFilterPolicyStatus) DeepCopy() *HTTPFilterPolicyStatus {
	if in == nil {
		return nil
	}
	out := new(HTTPFilterPolicyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPPlugin) DeepCopyInto(out *HTTPPlugin) {
	*out = *in
	in.Config.DeepCopyInto(&out.Config)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPPlugin.
func (in *HTTPPlugin) DeepCopy() *HTTPPlugin {
	if in == nil {
		return nil
	}
	out := new(HTTPPlugin)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceWebhook) DeepCopyInto(out *VirtualServiceWebhook) {
	*out = *in
	if in.VirtualService != nil {
		in, out := &in.VirtualService, &out.VirtualService
		*out = new(v1beta1.VirtualService)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceWebhook.
func (in *VirtualServiceWebhook) DeepCopy() *VirtualServiceWebhook {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceWebhook)
	in.DeepCopyInto(out)
	return out
}
