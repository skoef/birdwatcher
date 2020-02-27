# birdwatcher
[![Go Report Card](https://goreportcard.com/badge/github.com/skoef/birdwatcher)](https://goreportcard.com/report/github.com/skoef/birdwatcher)
> healthchecker for [BIRD](https://bird.network.cz/)-anycasted services

This project is heavily influenced by [anycast-healthchecker](https://github.com/unixsurfer/anycast_healthchecker). If you want to know more about use cases of birdwatcher, please read their excellent documention about anycasted services and how a healthchecker can contribute to a more stable service availability.

In a nutshell: birdwatcher periodically checks a specific service and tells BIRD which prefixes to announce or to withdraw when the service appears to be up or down respectively.

## Why birdwatcher
When I found out about anycast-healthchecker (sadly only recently on [HaproxyConf 2019](https://www.haproxyconf.com/)), I figured this would solve the missing link between BIRD and whatever anycasted services I have running (mostly haproxy though). Currently however in anycast-healthchecker, it is not possible to specify multiple prefixes to a service. Some machines in these kind of setups are announcing *many* `/32` and `/128` prefixes and I ended up specifiying so many services (one per prefix) that python crashed giving me a `too many open files` error. At first I tried to patch anycast-healthchecker but ended up writing something similar, hence birdwatcher.

It is written in Go because I like it and running multiple threads is easy.

## Example usage

This simple configuration enables managing IPv4 daemon of BIRD, runs `haproxy_check.sh` every second and manages 2 prefixes based on the exit code of the script:

```toml
[ipv4]
enable = true

[services]
  [services."foo"]
  command = "/usr/bin/haproxy_check.sh"
  prefixes = ["192.168.0.0/24", "192.168.1.0/24"]
  
```

Sample output in `/etc/bird/birdwatcher.conf` would be:

```
# DO NOT EDIT MANUALLY
function match_route()
{
	return net ~ [
		1.2.3.4/32,
		2.3.4.5/26,
		3.4.5.6/24,
		4.5.6.7/21
	];
}
```


## Configuration

**[ipv4]**
----------
Configuration section specific to manage the IPv4 BIRD daemon, `bird`.

|key          |description|
|-------------|-----------|
|enable       |Whether or not to manage IPv4 daemon of BIRD. Note that at least on of both ipv4 or ipv6 should be enabled for birdwatcher to start. Defaults to **false**.|
|configfile   |Path to configuration file that will be generated and should be included in the BIRD configuration. Defaults to **/etc/bird/birdwatcher.conf**.|
|reloadcommand|Command to invoke to signal BIRD the configuration should be reloaded. Defaults to **/usr/sbin/birdc configure**.|


**[ipv6]**
----------
Configuration section specific to manage the IPv6 BIRD daemon, `bird6`.

|key          |description|
|-------------|-----------|
|enable       |Whether or not to manage IPv6 daemon of BIRD. Note that at least on of both ipv4 or ipv6 should be enabled for birdwatcher to start. Defaults to **false**.|
|configfile   |Path to configuration file that will be generated and should be included in the BIRD configuration. Defaults to **/etc/bird/birdwatcher6.conf**.|
|reloadcommand|Command to invoke to signal BIRD the configuration should be reloaded. Defaults to **/usr/sbin/birdc6 configure**.|


**[services]**
--------------
Each service under this section can have the following settings:

|key          |description|
|-------------|-----------|
|command      |Command that will be periodically run to check if the service should be considered up or down. The result is based on the exit code: a non-zero exit codes makes birdwatcher decide the service is down, otherwise it's up. **Required**|
|functionname |Specify the name of the function birdwatcher will generate. You can use this function name to use in your protocol export filter in BIRD. Defaults to **match_route**.|
|interval     |The interval in seconds at which birdwatcher will check the service. Defaults to **1**|
|timeout      |Time in seconds in which the check command should complete. Afterwards it will be handled as if the check command failed|
|fail         |The amount of times the check command should fail before the service is considered to be down. Defaults to **1**|
|rise         |The amount of times the check command should succeed before the service is considered to be up. Defaults to **1**|
|prefixes     |Array of prefixes, mixed IPv4 and IPv6. At least 1 prefix is **required** per service|