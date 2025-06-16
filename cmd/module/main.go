//go:build linux

package main

import (
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"

	rftransmitter433mhz "rftransmitter433mhz"
)

func main() {
	// ModularMain can take multiple APIModel arguments, if your module implements multiple models.
	module.ModularMain(resource.APIModel{API: generic.API, Model: rftransmitter433mhz.RfTransmitter})
}
