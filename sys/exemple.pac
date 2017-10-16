function FindProxyForURL(url, host)
{
var proxy = "PROXY 192.168.0.10:3128; PROXY 192.168.0.20:3128; DIRECT" ;
var site = dnsResolve(host);

if (isInNet(site, "10.0.0.0", "255.0.0.0")) {
            return "DIRECT";
}

if (isInNet(site, "172.16.0.0", "255.240.0.0")) {
            return "DIRECT";
}

if (isInNet(site, "192.168.0.0", "255.255.0.0")) {
            return "DIRECT";
}

if (isInNet(site, "127.0.0.0", "255.255.255.0")) {
            return "DIRECT";
}

if (dnsDomainIs(host,"www.example.com")) {
            return "DIRECT";
}

return proxy ;

}
