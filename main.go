package main

import (
	"errors"
	"os"
	"os/signal"

	"github.com/bosh-loki/firehose-loki-client/lokiclient"
	"github.com/bosh-loki/firehose-loki-client/lokifirehosenozzle"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	apiEndpoint = kingpin.Flag(
		"api.endpoint", "Cloud Foundry API Endpoint ($FIREHOSE_API_ENDPOINT)",
	).Envar("FIREHOSE_API_ENDPOINT").Required().String()

	apiUsername = kingpin.Flag(
		"api.username", "Cloud Foundry API Username ($FIREHOSE_API_USERNAME)",
	).Envar("FIREHOSE_API_USERNAME").Required().String()

	apiPassword = kingpin.Flag(
		"api.password", "Cloud Foundry API Password ($FIREHOSE_API_PASSWORD)",
	).Envar("FIREHOSE_API_PASSWORD").Required().String()

	skipSSLValidation = kingpin.Flag(
		"skip-ssl-verify", "Disable SSL Verify ($FIREHOSE_SKIP_SSL_VERIFY)",
	).Envar("FIREHOSE_SKIP_SSL_VERIFY").Default("false").Bool()
)

type LokiAdapter struct {
	client *lokiclient.Client
}

func main() {
	log.AddFlags(kingpin.CommandLine)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	cfConfig := &cfclient.Config{
		ApiAddress:        *apiEndpoint,
		Username:          *apiUsername,
		Password:          *apiPassword,
		SkipSslValidation: *skipSSLValidation}

	cfClient, err := cfclient.NewClient(cfConfig)
	if err != nil {
		log.Fatal(err)
	}

	client := lokifirehosenozzle.NewLokiFirehoseNozzle(cfConfig, cfClient, "loki")

	firehose, errorhose := client.Connect()
	if firehose == nil {
		panic(errors.New("firehose was nil"))
	} else if errorhose == nil {
		panic(errors.New("errorhose was nil"))
	}
	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt)

	for {
		select {
		case envelope := <-firehose:
			if envelope == nil {
				log.Errorln("received nil envelope")
			} else {
				client.PostToLoki(envelope)
			}
		case err := <-errorhose:
			if err == nil {
				log.Errorln("received nil envelope")
			} else {
				log.Errorln(err)
			}
		case <-exitSignal:
			os.Exit(0)
		}
	}
}