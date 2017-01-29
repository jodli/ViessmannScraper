package main

import (
  "fmt"
  "net"
  "bufio"
  "os"
  "flag"
  "time"
)

const MAX_RECONNECT = 5
const DIAL_TIMEOUT = 5 * time.Second

var address string
var port int

type Vclient struct {
  connected bool

  connection net.Conn
  reader *bufio.Reader
  writer *bufio.Writer
}

var client Vclient

func init() {
  flag.StringVar(&address, "address", "raspberrypi-2", "The address of the vcontrold telnet server.")
  flag.IntVar(&port, "port", 3002, "The port of the vcontrold telnet server.")

  client = Vclient{connected: false}
}

func Connect() bool {
  if client.connected {
    return client.connected
  }

  addressStr := fmt.Sprintf("%s:%d", address, port)
  connTmp, err := net.DialTimeout("tcp", addressStr, DIAL_TIMEOUT)
  
  if err != nil {
    fmt.Println(err)
    client.connected = false
    return client.connected
  }

  client.connected = true
  client.connection = connTmp
  fmt.Println("Connected to:", connTmp.RemoteAddr())

  return client.connected
}

func Write(cmd string) bool {
  return true
}

func setup() {
  for i := 0; i < MAX_RECONNECT; i++ {
    fmt.Println("Attempt to connect", i+1, "of", MAX_RECONNECT, "tries.")
    if Connect() {
      client.reader = bufio.NewReader(client.connection)
      client.writer = bufio.NewWriter(client.connection)
      break
    }
    time.Sleep(5 * time.Second)
  }

  if !client.connected {
    // If we are still not connected at this point we exit and eventually restart the container.
    os.Exit(1)
  }
}

func main() {
  flag.Parse()
  fmt.Println("ViessmannScraper")

  setup()

  for {
    fmt.Println("Reading...")
    str, err := client.reader.ReadString('\n')

    if len(str) > 0 {
      fmt.Printf("Received %d bytes: %s", len(str), str)
    }
    if err != nil {
      break
    }
  }
}
