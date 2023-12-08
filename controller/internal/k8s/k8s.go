package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetObjectKey(obj *metav1.ObjectMeta) string {
	return obj.Namespace + "/" + obj.Name
}
