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

var connected bool = false
var conn net.Conn

func init() {
  flag.StringVar(&address, "address", "raspberrypi-2", "The address of the vcontrold telnet server.")
  flag.IntVar(&port, "port", 3002, "The port of the vcontrold telnet server.")
}

func Connect() bool {
  if connected {
    return true
  }
  addressStr := fmt.Sprintf("%s:%d", address, port)

  connTmp, err := net.DialTimeout("tcp", addressStr, DIAL_TIMEOUT)
  
  if err != nil {
    fmt.Println(err)
    connected = false
    return false
  }

  connected = true
  conn = connTmp
  fmt.Println("Connected to:", conn.RemoteAddr())
  return true
}

func main() {
  flag.Parse()
  fmt.Println("ViessmannScraper")

  for i := 0; i < MAX_RECONNECT; i++ {
    fmt.Println("Attempt to connect", i+1, "of", MAX_RECONNECT, "tries.")
    if Connect() {
      break
    }
    time.Sleep(1 * time.Second)
  }

  if !connected {
    os.Exit(1)
  }

  fmt.Println("Writing to server...")
  fmt.Fprintf(conn, "device\r\n")

  connbuf := bufio.NewReader(conn)
  for {
    fmt.Println("Reading...")
    str, err := connbuf.ReadString('\n')

    if len(str) > 0 {
      fmt.Printf("Received %d bytes: %s", len(str), str)
    }
    if err != nil {
      break
    }
  }
}
