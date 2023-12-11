package model

import (
	"k8s.io/apimachinery/pkg/types"
)

type Gateway struct {
	NsName *types.NamespacedName
	Port   uint32
}

type VirtualHost struct {
	Gateway *Gateway
	Name    string
}

type GoPluginConfig struct {
	Config interface{} `json:"config"`
}
