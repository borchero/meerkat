package ovpn

import (
	"strconv"
	"strings"

	api "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
)

// ParseRoutes is a utility function to convert the api's subnet masks into routes for the OVPN
// config file.
func ParseRoutes(subnets []api.SubnetMask) []ConfigRoute {
	result := make([]ConfigRoute, len(subnets))
	for i, subnet := range subnets {
		splits := strings.Split(string(subnet), "/")
		result[i].IP = splits[0]
		result[i].Mask = getMask(splits[1])
	}
	return result
}

// ParseRoutesString is a utility function to convert the api's subnet masks into iptable routes.
func ParseRoutesString(subnets []api.SubnetMask) []string {
	result := make([]string, len(subnets))
	for i, subnet := range subnets {
		splits := strings.Split(string(subnet), "/")
		result[i] = splits[0] + "/" + getMask(splits[1])
	}
	return result
}

func getMask(stringSize string) string {
	size, err := strconv.Atoi(stringSize)
	if err != nil {
		panic(err)
	}
	mask := 0xFFFFFFFF << (32 - size)
	limbs := []string{
		strconv.Itoa((mask >> 24) & 0xFF),
		strconv.Itoa((mask >> 16) & 0xFF),
		strconv.Itoa((mask >> 8) & 0xFF),
		strconv.Itoa(mask & 0xFF),
	}
	return strings.Join(limbs, ".")
}
