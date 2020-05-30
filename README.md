![](imgs/interlock-scheme.png?raw=true)

Interlock is a DNS failover and management tool based on Cloudflare APIs, the main goal of the project is to exclude CDN/site origins relying on HTTP Response codes or latency

Here you can find some use-cases:
- HA on Websites served by multiple datacenters (t.g. static sites, DR sites...)
- Multi-CDN origins management
- Measuring HTTP endpoints latency on InfluxDB

Some example setups are briefly described below:
- Public CI/CD scheduled container runs in order to remotely check that your origins are healthy (both latency and response)
- Private CI/CD, in order to run locally and check service response e.g. (private services reachable via VPNs\same LAN segment)
- External backup failover in order to have at least one site responding when SHTF and you cannot change rapidly DNS (e.g. Maintenance page hostend on a third-party origin included in the conf responding on the same domain)
- Node on your server LAN do measure latency in order to notify you if your site is getting slower (running on Dry Run = No CF records changes)

Before running in any production env be sure to understand the code underneath it (**Security notice**, you could **DELETE all** your DNS records if configured badly)

## Demo
[![asciicast ](https://asciinema.org/a/335086.png)](https://asciinema.org/a/335086)

Sample Grafana dashboard build with latency response data gathered via interlock

![](imgs/interlock-grafana.png?raw=true)

## Requirements
- Cloudflare account (also free tier works)
- Docker or GitlabCI or Golang


## Setup variables
Before running the code you will require to securely set these ENV variables:
- CF_API (**Mandatory** - You can generate a global API key from Cloudflare account settings)
- CF_EMAIL (**Mandatory** - Your Cloudflare account email)
- TGBOT_TOKEN  (Optional - Required only for Telegram Notifications , Create a TG Bot and use the bot token as a sender)
- TGBOT_CHATID (Optional - Required only for Telegram Notifications , Your private TG chat id associated to your phone number)
- INFLUXDB_PASSWORD (Optional - Required only for InfluxDB Metrics, other settings are inside interlock.conf)
- DRYRUN (Optional - disables DNS changes)

Security notice: Do not save ENV vars on public CI/CDs if not masked or are you sure about it and you know what are you doing 

## Configuration
You can find an example configuration here for interlock.conf
```
Origins = [
  "example.org http://1.2.3.4",
  "example.org http://1.2.3.5",
  "example.com http://1.2.3.5"
]
MaxLatency = 150
InfluxdbHost = "http://influxdb_host.example.org:8086"
InfluxdbDatabase = "database_interlock"
InfluxdbUsername = "influxdb_user"
```

MaxLatency parameter is used to exclude origins based on http response latency

Influx variables are used to send metrics to an InfluxDB

With this simple configuration the tool will check on every run if the sites under Origins are responding to the URIs specified, other than this pure HTTP response code check, latency will be measured (if MaxLatency > 0).

## Quick-start

In order to run interlock via Docker you will require to build your local image and run it with the following options:
```
# docker build -t interlock .
# docker run  -e TGBOT_TOKEN -e TGBOT_CHATID -e CF_API -e CF_EMAIL -e INFLUXDB_PASSWORD interlock
```

To run interlock under Golang you will require to download all imported code libraries and then execute the code (as always with ENV setup)
```
# go get .
# go run interlockd.go
Checking sami.pw on IP http://185.199.108.153...Sending metrics to influxdb  http://influxdbhost:8086
Latency is OK  144
Origin OK...CF Record OK

Checking sami.pw on IP http://185.199.109.153...Sending metrics to influxdb  http://influxdbhost:8086
Latency is OK  36
Origin OK...CF Record OK

Checking sa.mi.it on IP http://185.199.110.153...Sending metrics to influxdb  http://influxdbhost:8086
Latency is OK  37
Origin OK...CF Record OK

Checking sa.mi.it on IP http://185.199.111.153...Sending metrics to influxdb  http://influxdbhost:8086
Latency is OK  37
Origin OK...CF Record OK
```

## Running as a k8s CronJob

Before running into the .yml code you will require an image of your configuration pushed into a private/public registry (e.g. Gitlab or DockerHub), once ready you can go on the next steps.

In order to run interlock on a k8s cluster you will require to use the same environment variables passed inside the .yml file, you can edit and run as the example below:
```
# kubectl apply -f k8s/interlock-deployment.yml
# kubectl  get po | grep interlock # to check if pods created via CronJob are completed
interlockd-1590766200-przlb                        0/1     Completed   0          12m
interlockd-1590766500-d2hsw                        0/1     Completed   0          7m49s
interlockd-1590766800-dc4qp                        0/1     Completed   0          2m56s
# kubectl  describe pods | grep interl -A3 -B4  # in order to troubleshoot image pull issues

```
