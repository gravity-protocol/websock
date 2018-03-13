/*
 * Copyright (c) 2017 AlexRuzin (stan.ruzin@gmail.com)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package websock

import (
    "testing"
    "errors"
    "strconv"

    "github.com/AlexRuzin/util"
    "encoding/json"
    "io/ioutil"
)

/*
 * Configuration file name
 */
const JSON_FILENAME                 string = "config.json"

/*
 * Servertype enumerator
 */
type serverType uint8
const (
    TYPE_NONE                       serverType = iota
    TYPE_SERVER                     /* Type is a server */
    TYPE_CLIENT                     /* Type is a client */
)

/*
 * For example, the config.json file uses the following key/value structure:
 *
 * {
 *   // true -> server/listener mode, false -> client/connect mode
 *   "Server": true,
 *
 *   // Debug is piped to stdout
 *   "Verbosity": true,
 *
 *   // Encryption/compression settings
 *   "Encryption": true,
 *   "Compression": true,
 *
 *   // Connectivity settings for both client and server
 *   "Port": 2222,
 *   "Path": "/gate.php",
 *   "Domain": "127.0.0.1",
 *
 *   // Do not change this setting
 *   "ModuleName": "websock"
 * }
 */
const moduleName                    string = "websock"
type configInput struct {
    /* Default test mode */
    Server                          bool

    Verbosity                       bool

    Encryption                      bool
    Compression                     bool

    Port                            uint16
    Path                            string
    Domain                          string

    ModuleName                      string
}

var (
    genericConfig                   *configInput = nil
    mainServer                      *NetChannelService = nil
)
func TestMainChannel(t *testing.T) {
    /* Parse the user input and create a configInput instance */
    config, _ := func () (*configInput, error) {
        /* Read in the configuration file `config.json` */

        rawFile, err := ioutil.ReadFile(JSON_FILENAME)
        if err != nil {
            panic(err)
        }

        /*
         * Build the configInput structure
         */
        var (
            output                  configInput
            parseStatus             error = nil
        )
        parseStatus = json.Unmarshal(rawFile, &output)
        if parseStatus != nil {
            panic(parseStatus)
        }
        if output.ModuleName != moduleName {
            panic(errors.New("invalid configuration file: " + JSON_FILENAME))
        }

        return &output, nil
    } ()
    genericConfig = config

    /* It is absolutely required to use encryption, therefore check for this prior to anything futher */
    if config.Encryption == false {
        panic(errors.New("must use the 'encrypt' flag to 'true'"))
    }

    if config.Verbosity == true {
        func(config *configInput) {
            switch config.Server {
            case false: /* Client mode */
                D("We are running in TYPE_CLIENT mode. Default target server is: " + "http://" +
                    config.Domain + ":" + util.IntToString(int(config.Port)) +
                        config.Path)
                break
            case true: /* Server mode */
                D("We are running in TYPE_SERVER mode. Default listening port is: " +
                    util.IntToString(int(config.Port)))
                D("Default listen path is set to: " + config.Path)
            }

            D("Using encryption [forced]: " + strconv.FormatBool(config.Encryption))
            D("Using compression [optional]: " + strconv.FormatBool(config.Compression))
        }(config)
    }

    switch config.Server {
    case false: /* Client mode */
        var gateURI string = "http://" + config.controllerAddress + ":" +
            util.IntToString(int(config.controllerPort)) + config.controllerGatePath
        D("Client target URI is: " + gateURI)

        client, err := BuildChannel(gateURI /* Primary URI (scheme + domain + port + path) */ ,

            /* The below inlines will determine which flags to use based on use input */
            func(useDebug bool) FlagVal {
                if useDebug == true {
                    return FLAG_DEBUG
                }

                return 0
            }(config.verbosity)|
                func(useEncryption bool) FlagVal {
                    if useEncryption == true {
                        return FLAG_ENCRYPT
                    }

                    return 0
                }(config.useEncryption)|
                func(useCompression bool) FlagVal {
                    if useCompression == true {
                        return FLAG_COMPRESS
                    }

                    return 0
                }(config.useCompression),
        )
        if err != nil {
            panic(err)
        }
        if err := client.InitializeCircuit(); err != nil {
            panic(err)
        }

        break
    case true: /* Server mode */
        D("Server is running on localhost, port: " + util.IntToString(int(config.controllerPort)) +
            ", on HTTP URI path: " + config.controllerGatePath)

        server, err := CreateServer(config.controllerGatePath, config.controllerPort,
            /* The below inlines will determine which flags to use based on use input */
            func(useDebug bool) FlagVal {
                if useDebug == true {
                    return FLAG_DEBUG
                }

                return 0
            }(config.verbosity)|
                func(useEncryption bool) FlagVal {
                    if useEncryption == true {
                        return FLAG_ENCRYPT
                    }

                    return 0
                }(config.useEncryption)|
                func(useCompression bool) FlagVal {
                    if useCompression == true {
                        return FLAG_COMPRESS
                    }

                    return 0
                }(config.useCompression),
            incomingClientHandler)
        if err != nil {
            panic(err)
        }
        mainServer = server
    }

    /* Wait forever */
    util.WaitForever()
}

func incomingClientHandler(client *NetInstance, server *NetChannelService) error {
    util.SleepSeconds(14)
    client.Write([]byte("some random data"))

    util.SleepSeconds(25)
    if client.Len() != 0 {
        data := make([]byte, client.Len())
        client.Read(data)
        util.DebugOut(string(data))
    }
    return nil
}

func D(debug string) {
    if genericConfig.verbosity == true {
        util.DebugOut("[+] " + debug + "\r\n")
    }
}

func T(debug string) {
    util.ThrowN("[!] " + debug)
}
