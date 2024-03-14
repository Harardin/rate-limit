package consul

type GetServiceAddressResponseItem struct {
	Address string
	Port    int
}

type GetServiceAddressResponse []GetServiceAddressResponseItem
