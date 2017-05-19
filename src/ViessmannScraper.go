package main

import (
  "fmt"
  "net"
  "bufio"
  "os"
  "flag"
  "time"
  "strings"
  "strconv"
)

const MAX_RECONNECT = 5
const DIAL_TIMEOUT = 5 * time.Second
const POLLING_RATE = 30 * time.Second

const (
  TEMP_RUECKLAUF = "getTempRuecklauf"
  TEMP_ABGAS = "getTempAbgas"
  TEMP_SOLAR_DACH = "getTempSolarDach"
  TEMP_SOLAR_WW = "getTempSolarWW"
  TEMP_SPEICHER = "getTempSpeicher"
  TEMP_SOLL_WW = "getTempSollWW"
  TEMP_AUSSEN_GEDAEMPFT = "getTempAussenGedaempft"
  TEMP_AUSSEN_GEMISCHT = "getTempAussenGemischt"
  TEMP_IST_KESSEL = "getTempIstKessel"
  TEMP_SOLL_KESSEL = "getTempSollKessel"

  STATUS_SOLAR = "getStatusSolar"
  STATUS_PUMPE_SPEICHERLADE = "getStatusSpeicherlade"
  STATUS_PUMPE_HEIZKREIS_A1 = "getStatusHeizkreis_A1"
  STATUS_PUMPE_HEIZKREIS_M1 = "getStatusHeizkreis_M1"
  STATUS_PUMPE_ZIRKULATION = "getStatusZirkulation"
  STATUS_RELAIS_K12 = "getStatusRelaisK12"
  STATUS_PUMPE_INTERN = "getStatusIntern"
  STATUS_FLOW_SWITCH = "getStatusFlowSwitch"

  MISC_STARTS_BRENNER = "getStartsBrenner"
  MISC_LAUFZEIT_BRENNER = "getStundenBrenner"
  MISC_LAUFZEIT_BRENNER_STUFE1 = "getStundenBrenner1"
  MISC_LAUFZEIT_BRENNER_STUFE2 = "getStundenBrenner2"
  MISC_SAMMELSTOERUNG = "getSammelStoerung"
  MISC_Stoerung0 = "getStoerung0"
  MISC_TIME = "getTime"
)

var address string
var port int

type Vclient struct {
  connected bool

  connection net.Conn
  reader *bufio.Reader
  writer *bufio.Writer
  channel chan string
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

func Write(cmd string) {
  fmt.Println("Writing:", cmd)

  n, err := client.writer.WriteString(cmd + "\r\n")

  if n > 0 {
    fmt.Println("Wrote", n, "bytes.")
  }

  if err != nil {
    fmt.Println(err)
    client.connected = false
  }

  err = client.writer.Flush()

  if err != nil {
    fmt.Println(err)
    client.connected = false
  }
}

func Read() {
  fmt.Println("Starting read thread")
  for {
    str, err := client.reader.ReadString('\n')

    if len(str) > 0 {
      fmt.Print("Read:", str)
      client.channel <- str
    }

    if err != nil {
      fmt.Println(err)
      client.connected = false
      break
    }
  }
  fmt.Println("Stopping read thread")
}

func Process(commands []string) {
  fmt.Println("Starting process thread")
  for {
    fmt.Println("Starting new process cycle")

    for _, command := range commands {
      Write(command)

      str, ok := <- client.channel
      if !ok {
        fmt.Println("Channel closed")
        break
      }

      fmt.Print("Processing: ", str)

      str = strings.Replace(str, "vctrld>", "", -1)
      str = strings.Replace(str, "\n", "", -1)
      values := strings.SplitN(str, " ", 2)
      ParseValues(command, values)
    }

    fmt.Println("Sleeping for", POLLING_RATE)
    time.Sleep(POLLING_RATE)
  }
  fmt.Println("Stopping process thread")
}

func ParseValues(command string, values []string) {
  fmt.Println("Parsing command:", command)
  fmt.Println("with values:")
  for _, value := range values {
    fmt.Println(value)
  }

  if strings.Contains(command, "Temp") {
    floatValue, err := strconv.ParseFloat(values[0], 32)
    if err != nil {
      fmt.Println(err)
    } else {
      fmt.Println("============>", time.Now(), command, ":", floatValue)
    }
  } else if strings.Contains(command, "Status") {
    boolValue, err := strconv.ParseBool(values[0])
    if err != nil {
      fmt.Println(err)
    } else {
      fmt.Println("============>", time.Now(), command, ":", boolValue)
    }
  }
}

func setup() {
  for i := 0; i < MAX_RECONNECT; i++ {
    fmt.Println("Attempt to connect", i+1, "of", MAX_RECONNECT, "tries.")
    if Connect() {
      client.reader = bufio.NewReader(client.connection)
      client.writer = bufio.NewWriter(client.connection)

      client.channel = make(chan string)
      break
    }
    time.Sleep(5 * time.Second)
  }

  if !client.connected {
    // If we are still not connected at this point we exit and eventually restart the container.
    fmt.Println("Could not connect to vcontrold server. Exiting...")
    os.Exit(1)
  }
}

func main() {
  flag.Parse()
  fmt.Println("======== ViessmannScraper ========")

  var commands []string
  commands = append(commands, TEMP_RUECKLAUF, TEMP_ABGAS, TEMP_SOLAR_DACH,
                    TEMP_SOLAR_WW, TEMP_SPEICHER, TEMP_SOLL_WW, TEMP_AUSSEN_GEDAEMPFT,
                    TEMP_AUSSEN_GEMISCHT, TEMP_IST_KESSEL, TEMP_SOLL_KESSEL)
  commands = append(commands, STATUS_SOLAR, STATUS_PUMPE_SPEICHERLADE, STATUS_PUMPE_HEIZKREIS_A1,
                    STATUS_PUMPE_HEIZKREIS_M1, STATUS_PUMPE_ZIRKULATION, STATUS_RELAIS_K12,
                    STATUS_PUMPE_INTERN, STATUS_FLOW_SWITCH)
  commands = append(commands, MISC_STARTS_BRENNER, MISC_LAUFZEIT_BRENNER, MISC_LAUFZEIT_BRENNER_STUFE1,
                    MISC_LAUFZEIT_BRENNER_STUFE2, MISC_SAMMELSTOERUNG, MISC_Stoerung0, MISC_TIME)

  for {
    if !client.connected {
      setup()

      // We suppose the read thread was broken so we start it again.
      go Read()
      go Process(commands)
    }
    time.Sleep(10 * time.Second)
  }
}
