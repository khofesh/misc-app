package main

import (
	"flag"
	"net/url"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	decodeFlag := flag.Bool("decode", false, "decode the input file")
	encodeFlag := flag.Bool("encode", false, "encode the input file")
	flag.Parse()

	// zerolog
	multiWriters := zerolog.MultiLevelWriter(os.Stdout)
	log.Logger = zerolog.New(multiWriters).With().Timestamp().Logger()

	if (*decodeFlag && *encodeFlag) || (!*decodeFlag && !*encodeFlag) {
		log.Error().Msg("error: you must specify exactly one operation (--docode OR --encode)")
		flag.Usage()
		os.Exit(1)
	}

	var inputFile, outputFile string

	if *decodeFlag {
		inputFile = "input/decode.txt"
		outputFile = "output/decoded.txt"
	} else if *encodeFlag {
		inputFile = "input/encode.txt"
		outputFile = "output/encoded.txt"
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		log.Err(err).Msgf("error reading input file %s: %v\n", inputFile, err)
		os.Exit(1)
	}

	inputString := string(data)
	var result string

	// process
	if *decodeFlag {
		result, err = url.QueryUnescape(inputString)
		if err != nil {
			log.Err(err).Msgf("Error decoding string: %v\n", err)
			os.Exit(1)
		}
		log.Info().Msg("decoded successfully")
	} else if *encodeFlag {
		result = url.QueryEscape(inputString)
		log.Info().Msg("encoded successfully")
	}

	err = os.WriteFile(outputFile, []byte(result), 0644)
	if err != nil {
		log.Err(err).Msgf("error writing to output file %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	log.Info().Msgf("operation completed. result saved to %s\n", outputFile)
}
