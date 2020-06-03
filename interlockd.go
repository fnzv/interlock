package main

import (
	"fmt"
	"time"
	"net"
	"net/url"
	"crypto/tls"
	"os"
	"log"
	"strings"
	"net/http"
	"bytes"

	"github.com/tcnksm/go-httpstat"
	"github.com/cloudflare/cloudflare-go"
	"github.com/BurntSushi/toml"
	"github.com/influxdata/influxdb1-client/v2"
)


func influxDBClient() client.Client {
    c, err := client.NewHTTPClient(client.HTTPConfig{
        Addr:     influxdb_host,
        Username: influxdb_username,
		Password: influxdb_password,
    })
    if err != nil {
        log.Fatalln("Error: ", err)
    }
    return c
}

func createMetrics(c client.Client, domain string, latency int,uri string) {
    bp, err := client.NewBatchPoints(client.BatchPointsConfig{
        Database:  influxdb_database,
        Precision: "s",
    })

    if err != nil {
        log.Fatalln("Error: ", err)
    }

    eventTime := time.Now().Add(time.Second * -20)

            tags := map[string]string{
                "site": domain,
                "origin": uri,
            }

            fields := map[string]interface{}{
				"domain": domain,
                "latency": latency,
            }

            point, err := client.NewPoint(
                "interlockd_response",
                tags,
                fields,
                eventTime.Add(time.Second*10),
            )
            if err != nil {
                log.Fatalln("Error: ", err)
            }

            bp.AddPoint(point)


    err = c.Write(bp)
    if err != nil {
        log.Fatal(err)
    }
}




func telegramNotify(msg string) {


	if os.Getenv("TGBOT_TOKEN") != "" && os.Getenv("TGBOT_CHATID") != "" {
    url := "https://api.telegram.org/"+os.Getenv("TGBOT_TOKEN")+"/sendMessage"
	var jsonStr = []byte(`{"chat_id": `+os.Getenv("TGBOT_CHATID")+`, "text": "`+msg+`", "disable_notification": true}`)
	
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
} else {

	fmt.Println("Telegram env vars are not set... skipping notification")
}
}



var site string

var ms_request int



func checkSiteAlive(site string,uri string) string {



	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		log.Println("Site unreachable, error: ")
		telegramNotify("Origin is unreachable on "+site)

		return "KO"
	}

	
    // This transport is what's causing unclosed connections.
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	
// Create a httpstat powered context
var result httpstat.Result
ctx := httpstat.WithHTTPStat(req.Context(), &result)
req = req.WithContext(ctx)


	hc := &http.Client{Timeout: 2 * time.Second, Transport: tr}
	

	req.Host = site
	//client := &http.Client{}
	resp, err := hc.Do(req)
	
    // Show the results
  //  log.Printf("\nDNS lookup: %d ms", int(result.DNSLookup/time.Millisecond))
  //  log.Printf("TCP connection: %d ms", int(result.TCPConnection/time.Millisecond))
  //  log.Printf("TLS handshake: %d ms", int(result.TLSHandshake/time.Millisecond))
  //  log.Printf("Server processing: %d ms", int(result.ServerProcessing/time.Millisecond))
//	log.Printf("Content transfer: %d ms", int(result.ContentTransfer(time.Now())/time.Millisecond))
	ms_request=int(result.ServerProcessing/time.Millisecond)

 if influxdb_host != "" && influxdb_password != "" && influxdb_username != "" {
   // sent to telegraf db domain - latency - ts
   fmt.Println("Sending metrics to influxdb ",influxdb_host)
   c := influxDBClient()
	createMetrics(c,site,ms_request,uri)
}

	if  resp == nil {
		fmt.Print("Origin NOPE...")
		//telegramNotify("Origin is responding Nope on "+site+" - "+uri)
        
		return "KO"
	}
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		if max_latency > 0 && ms_request < max_latency {
			fmt.Println("Latency is OK ",ms_request)
		return "OK"

		}
		if max_latency > 0 && ms_request > max_latency {
			fmt.Println("Latency is above configured treshold ****************************** ",ms_request)
			return "KO"
		} 
		if max_latency == 0 { 
			return "OK"
		}
	} 
    return "Nope"
}
var (
	origins []string
	dryrun string
	ip string
	max_latency int
	influxdb_host string
	influxdb_database  string
    influxdb_username string
    influxdb_password string 

	)
//	/ Info from config file
type Config struct {
			Origins []string
			MaxLatency int
			InfluxdbHost string
 			InfluxdbDatabase  string
			InfluxdbUsername string 
	}

	func ReadConfig() Config {
        var configfile = "interlockd.conf"
        _, err := os.Stat(configfile)
        if err != nil {
                log.Fatal("Config file is missing: ", configfile)
        }

        var config Config
        if _, err := toml.DecodeFile(configfile, &config); err != nil {
                log.Fatal(err)
        }
        //log.Print(config.Index)
        return config
}


func main() {
	// Construct a new API object
	if os.Getenv("CF_API") == ""  {
		fmt.Println("Cloudflare env vars are not set... aborting soon")
	}
	api, err := cloudflare.New(os.Getenv("CF_API"),os.Getenv("CF_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

var config = ReadConfig()
origins = config.Origins
dryrun = os.Getenv("DRYRUN")
max_latency = config.MaxLatency
influxdb_host = config.InfluxdbHost
influxdb_database = config.InfluxdbDatabase
influxdb_username = config.InfluxdbUsername
influxdb_password = os.Getenv("INFLUXDB_PASSWORD")

i:= 0
  for range origins {
    s := strings.Split(origins[i], " ")
	domain, uri := s[0], s[1]

	s1 := strings.Split(origins[i], "//")
	clean_domain := s1[1]
	ip_dns, err := net.LookupHost(clean_domain)
	if err != nil {
	  fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
	ip = ip_dns[0]
	fmt.Println("")
	fmt.Print("Checking "+domain+" on IP "+uri+"...")
	 i++

	 siteAlive:= checkSiteAlive(domain,uri)
	if ( siteAlive == "OK") {
		  // Fetch the zone ID
    fmt.Print("Origin OK...")
	id, err := api.ZoneIDByName(domain) // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		log.Fatal(err)
	}



				r := cloudflare.DNSRecord{
					Type:    "A",
					Name:    domain,
					Content: ip,
					Proxied: true,
					TTL:     1,
				}

				if dryrun == "" {

				resp, err := api.CreateDNSRecord(id, r) 
				
				 
	 			if strings.Contains(fmt.Sprintf("%+v\n", resp), "Success:true") {
					fmt.Println("Creating record ",r.Content,domain)
				    telegramNotify(fmt.Sprintf("Creating record %s on zone %s",r.Content,domain))

				} 				
				if err != nil && strings.Contains(err.Error(), "400") {
					fmt.Println("CF Record OK")
				}
				if err != nil && !(strings.Contains(err.Error(), "400") ){
					fmt.Printf("Error: %v\n", err.Error())
				
				}
			}
				recs, err := api.DNSRecords(id, cloudflare.DNSRecord{Type: "A", Name: domain})

/// Check if any existing record doesn't match the working ones
if err == nil && len(recs) > 0 {
	k:= 0
	for range recs {
	//r := recs[k]
			//	//r.Content = ip

	ip = strings.Trim(strings.Trim(uri, "https://") , "http://")

	//delete:=0
	//j:=0
//	for range origins {


	    //concatenanted_conf:=strings.Join(origins[:], ",")
		//if (!strings.Contains(concatenanted_conf, r.Content) ) {
		//	fmt.Println("IP doesnt exists  ",r.Content)
		//	fmt.Sprintf("Deleting record %s on %s ",r.Content,domain)
		//	if dryrun == "" {
		//	err = api.DeleteDNSRecord(id, r.ID)
		//		telegramNotify(fmt.Sprintf("Deleting record %s on %s ",r.Content,domain))
		//	if err != nil {
		//			fmt.Printf("Error: %v\n", err)
		//		return
		//	}
//
		//}
//
//
		//}
		//fmt.Println("check if  "+ip+" exist on conf")
	//	fmt.Println(s[1][i])
 		//if r.Content != ip {
			//	//r.Content = ip
		///r.Proxied = true
			//	err = api.UpdateDNSRecord(id, r.ID, r)
	//		fmt.Println("IP not equal to origin")
	//		delete++
		
	//	}


 //j++
//	}
    // concatena gli ip della conf in una stringa
	//se la conf non contiene l'ip remoto rimuovi l'ip remoto  


//	fmt.Println("Delete flag ",delete)

k++
	}

}


	  }
	if (siteAlive == "Nope" || siteAlive == "KO") {
	   id, err := api.ZoneIDByName(domain) // Assuming example.com exists in your Cloudflare account already
        
				recs, err := api.DNSRecords(id, cloudflare.DNSRecord{Type: "A", Name: domain} )
				j:=0
				for range recs { /// LOOP DNS  RECORDS ON CF
					if err == nil && len(recs) > 0 {
						r := recs[j]
						ip := strings.Trim(strings.Trim(uri, "https://") , "http://")

						if r.Content == ip {
							fmt.Sprintf("Deleting record %s on %s ",r.Content,domain)
			               if dryrun == "" {

							telegramNotify(fmt.Sprintf("Deleting record %s on %s ",r.Content,domain))
							err = api.DeleteDNSRecord(id, r.ID)
							if err != nil {
									fmt.Printf("Error: %v\n", err)
								return
					
								   }
								}
					
						}

						
						
					j++
					
					}
	

	}







  

}


  }

}