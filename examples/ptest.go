package main

// Just a test to ensure PortScheduler is working as expected

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/byxorna/citadel/scheduler"
	"github.com/citadel/citadel"
	"github.com/citadel/citadel/cluster"
)

type logHandler struct {
}

func (l *logHandler) Handle(e *citadel.Event) error {
	log.Printf("type: %s time: %s image: %s container: %s\n",
		e.Type, e.Time.Format(time.RubyDate), e.Container.Image.Name, e.Container.ID)

	return nil
}

func main() {
	engines := make([]*citadel.Engine, 0)
	cert, err := tls.LoadX509KeyPair("/Users/gabe/.boot2docker/certs/boot2docker-vm/cert.pem", "/Users/gabe/.boot2docker/certs/boot2docker-vm/key.pem")
	if err != nil {
		log.Fatal(err)
	}
	// read in our certs for the TLS connection
	certs := make([]tls.Certificate, 1)
	certs = append(certs, cert)
	// make sure we provide the CA cert as well
	rootCAPool := x509.NewCertPool()
	caf, err := os.Open("/Users/gabe/.boot2docker/certs/boot2docker-vm/ca.pem")
	if err != nil {
		log.Fatal(err)
	}
	cafInfo, _ := caf.Stat()
	caData := make([]byte, cafInfo.Size())
	if _, err := caf.Read(caData); err != nil {
		log.Fatal(err)
	}
	success := rootCAPool.AppendCertsFromPEM(caData)
	if !success {
		log.Fatal("Unable to load Root CA cert")
	}

	tlsConfig := &tls.Config{
		Certificates: certs,
		RootCAs:      rootCAPool,
	}

	// now initialize our engines (just the boot2docker one for now)
	collinsAssets := []string{"192.168.59.103"}
	//query collins for our engines
	for _, asset := range collinsAssets {
		e := &citadel.Engine{
			ID:     asset,
			Addr:   fmt.Sprintf("tcp://%s:2376", asset),
			Memory: 2048,
			Cpus:   4,
			Labels: []string{asset, "dev"},
		}
		//if err := e.Connect(nil); err != nil {
		if err := e.Connect(tlsConfig); err != nil {
			log.Fatal(err)
		}
		engines = append(engines, e)
	}

	c, err := cluster.New(scheduler.NewResourceManager(), engines...)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err := c.RegisterScheduler("persistent", &scheduler.PortScheduler{}); err != nil {
		log.Fatal(err)
	}

	if err := c.Events(&logHandler{}); err != nil {
		log.Fatal(err)
	}

	boundPort := citadel.Port{
		HostIp:        "0.0.0.0",
		Port:          6379,
		ContainerPort: 6379,
		Proto:         "tcp",
	}
	ports := []*citadel.Port{}
	ports = append(ports, &boundPort)
	image := &citadel.Image{
		Name:      "redis:latest",
		Memory:    256,
		Cpus:      0.4,
		BindPorts: ports,
		Type:      "persistent",
	}

	for i := 0; i < 2; i++ {
		container, err := c.Start(image, false)
		if err != nil {
			log.Printf("Unable to schedule container %d: %s\n", i, err)
		} else {
			log.Printf("Scheduled container %s\n", container.ID)
		}
	}

	containers := c.ListContainers(false)

	for _, ct := range containers {
		log.Printf("Killing %s\n", ct.ID)
		if err := c.Kill(ct, 9); err != nil {
			log.Printf("Unable to kill %s: %s\n", ct, err)
		}

		log.Printf("Removing %s\n", ct.ID)
		if err := c.Remove(ct); err != nil {
			log.Printf("Unable to remove %s: %s\n", ct, err)
		}
	}
	/*
			c1 := containers[0]

		  log.Printf("Booted container: %s\n",c1)
		  duration := time.Second * 20
		  log.Printf("Sleeping for %s seconds\n", duration)
		  time.Sleep(duration)

		  log.Printf("Killing %s\n",c1)
			if err := c.Kill(c1, 9); err != nil {
				log.Fatal(err)
			}

		  log.Printf("Removing %s\n",c1)
			if err := c.Remove(c1); err != nil {
				log.Fatal(err)
			}
	*/
}
