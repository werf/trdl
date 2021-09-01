package docker

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
)

const (
	checkingStartCode = iota
	processingStartCode
	processingDataAndCheckingStopCode
	processingStopCode
)

var (
	artifactsTarStartReadCode = []byte("1EA01F53E0277546E1B17267F29A60B3CD4DC12744C2FA2BF0897065DC3749F3")
	artifactsTarStopReadCode  = []byte("A2F00DB0DEE3540E246B75B872D64773DF67BC51C5D36D50FA6978E2FFDA7D43")
)

func DisplayFromImageBuildResponse(w io.Writer, response types.ImageBuildResponse) error {
	dec := json.NewDecoder(response.Body)
	for {
		var jm jsonmessage.JSONMessage
		if err := dec.Decode(&jm); err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("unable to decode message from docker daemon: %s", err)
		}

		if err := jm.Display(w, false); err != nil {
			return err
		}
	}
}

func ReadTarFromImageBuildResponse(tarWriter io.Writer, buildLogWriter io.Writer, response types.ImageBuildResponse) error {
	dec := json.NewDecoder(response.Body)
	currentState := checkingStartCode
	var codeCursor int
	var bufferedData []byte

	for {
		var jm jsonmessage.JSONMessage
		if err := dec.Decode(&jm); err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("unable to decode message from docker daemon: %s", err)
		}

		if jm.Error != nil {
			return jm.Error
		}

		var startReadCodeSuspectedBytes []byte

		msg := jm.Stream
		if msg != "" {
			for _, b := range []byte(msg) {
				switch currentState {
				case checkingStartCode:
					if b == artifactsTarStartReadCode[0] {
						currentState = processingStartCode
						codeCursor++

						startReadCodeSuspectedBytes = append(startReadCodeSuspectedBytes, b)
					} else if _, err := buildLogWriter.Write([]byte{b}); err != nil {
						return fmt.Errorf("build log writer failed: %s", err)
					}

				case processingStartCode:
					if b == artifactsTarStartReadCode[codeCursor] {
						if len(artifactsTarStartReadCode) > codeCursor+1 {
							codeCursor++

							startReadCodeSuspectedBytes = append(startReadCodeSuspectedBytes, b)
						} else {
							currentState = processingDataAndCheckingStopCode
							codeCursor = 0

							startReadCodeSuspectedBytes = nil
						}
					} else {
						currentState = checkingStartCode
						codeCursor = 0

						if _, err := buildLogWriter.Write(append(startReadCodeSuspectedBytes, b)); err != nil {
							return fmt.Errorf("build log writer failed: %s", err)
						}
						startReadCodeSuspectedBytes = nil
					}

				case processingDataAndCheckingStopCode:
					bufferedData = append(bufferedData, b)

					if b == artifactsTarStopReadCode[0] {
						currentState = processingStopCode
						codeCursor++
						continue
					}

					if _, err := tarWriter.Write(bufferedData); err != nil {
						return fmt.Errorf("tar writer failed: %s", err)
					}

					bufferedData = nil
				case processingStopCode:
					bufferedData = append(bufferedData, b)

					if b == artifactsTarStopReadCode[codeCursor] {
						if len(artifactsTarStopReadCode) > codeCursor+1 {
							codeCursor++
						} else {
							return nil
						}
					} else {
						currentState = processingDataAndCheckingStopCode
						codeCursor = 0
					}
				}
			}
		}
	}
}
