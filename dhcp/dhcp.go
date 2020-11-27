package dhcp

// Creating a DHCP OFFER class
// Please note that the first name of a variable should be capitalized in order to accessed from outside the package
type DHCP struct{
    Ip_offer string
    Default_gateway string
    Subnet_mask string
    Dns_address string
}
