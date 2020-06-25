const errorCodes = [{
    type:"hazard",
    message:`Domain potentially available for registration`,
    explanation: `This domain is not listed in its tld's zone files. If it
    is still registrable, records pointing to this domain are at risk.`
},
{
    type:"warning",
    message:`NS record with IPv4/6 address`,
    explanation:`RFC 1035 stIPulates that an NS record should point to an 
    authoritative hostname.`
},
{
    type:"warning",
    message:`NS record has a wildcard domain`,
    explanation: `DNS implementations for wildcard records (records beginning 
    with an '*') vary and can lead to inconsistent results.`
},
{
    type:"warning",
    message:`NS record points to a potential TLD`,
    explanation:`This NS record points to a hostname at the tld level, and is 
    likely not authoritative for a given domain.`
},
{
    type:"warning",
    message:`IP belongs to a public nameserver`,
    explanation:`This IP address belongs to a public nameserver, and is likely
    not authoritative for a given domain.`
},
{
    type:"warning",
    message:`IP is a part of private address space`,
    explanation:`This IP address is part of the 10.0.0.0/8, 172.16.0.0/12,
    192.168.0.0/16, 127.0.0.1, fc00::/7, fd00::/8, or ::1 address spaces. These
    address spaces are reserved for private, unique local, or loopback IPs and
    are not publicly routable.`
},
{
    type:"warning",
    message:`IP does not have an AS`,
    explanation:`This IP address does not have an AS and is not publicly
    routable.`
}]