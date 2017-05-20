package main

import (
  "fmt"
  "net"
  "bufio"
  "os"
  "log"
  "io"
  //"io/ioutil"
  "flag"
  "time"
  "strings"
  "strconv"

  "github.com/influxdata/influxdb/client/v2"
)

const MAX_RECONNECT = 5
const DIAL_TIMEOUT = 5 * time.Second
const POLLING_RATE = 60 * time.Second

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
  STATUS_PUMPE_HEIZKREIS_M2 = "getStatusHeizkreis_M2"
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

var (
  Trace *log.Logger
  Info *log.Logger
  Error *log.Logger
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

var viessmann Vclient
var influx client.Client

func Init(traceHandle io.Writer, infoHandle io.Writer, errorHandle io.Writer) {
  flag.StringVar(&address, "address", "raspberrypi-2", "The address of the vcontrold telnet server.")
  flag.IntVar(&port, "port", 3002, "The port of the vcontrold telnet server.")

  Trace = log.New(traceHandle, "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)
  Info = log.New(infoHandle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
  Error = log.New(errorHandle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

  viessmann = Vclient{connected: false}

  var err error
  influx, err = client.NewHTTPClient(client.HTTPConfig {
    Addr: os.Getenv("INFLUX_URL"),
    Username: os.Getenv("INFLUX_USER"),
    Password: os.Getenv("INFLUX_PASS"),
  })

  if err != nil {
    Error.Println("Could not connect to InfluxDB:", err)
    os.Exit(1)
  }
}

func Connect() bool {
  if viessmann.connected {
    return viessmann.connected
  }

  addressStr := fmt.Sprintf("%s:%d", address, port)
  connTmp, err := net.DialTimeout("tcp", addressStr, DIAL_TIMEOUT)

  if err != nil {
    Error.Println(err)
    viessmann.connected = false
    return viessmann.connected
  }

  viessmann.connected = true
  viessmann.connection = connTmp
  Info.Println("Connected to:", connTmp.RemoteAddr())

  return viessmann.connected
}

func Write(cmd string) {
  Trace.Println("Writing:", cmd)

  n, err := viessmann.writer.WriteString(cmd + "\r\n")

  if n > 0 {
    Trace.Println("Wrote", n, "bytes.")
  }

  if err != nil {
    Error.Println(err)
    viessmann.connected = false
  }

  err = viessmann.writer.Flush()

  if err != nil {
    Error.Println(err)
    viessmann.connected = false
  }
}

func Read() {
  Trace.Println("Starting read thread")
  for {
    str, err := viessmann.reader.ReadString('\n')

    if len(str) > 0 {
      Trace.Print("Read:", str)
      viessmann.channel <- str
    }

    if err != nil {
      Error.Println(err)
      viessmann.connected = false
      break
    }
  }
  Trace.Println("Stopping read thread")
}

func Process(commands []string) {
  Trace.Println("Starting process thread")
  for {
    Info.Println("Starting new process cycle...")
    bp, _ := client.NewBatchPoints(client.BatchPointsConfig {
      Database: "telegraf",
      Precision: "s",
    })

    for _, command := range commands {
      Write(command)

      str, ok := <- viessmann.channel
      if !ok {
        Trace.Println("Channel closed")
        break
      }

      Trace.Print("Processing: ", str)

      str = strings.Replace(str, "vctrld>", "", -1)
      str = strings.Replace(str, "\n", "", -1)
      values := strings.SplitN(str, " ", 2)
      point := ParseValues(command, values)

      if point != nil {
        Trace.Println("Adding point to batch.")
        bp.AddPoint(point)
        Trace.Println("Now contains", len(bp.Points()), "points.")
      }
    }

    Info.Println("Writing", len(bp.Points()) , "points to InfluxDB.")
    err := influx.Write(bp)
    if err != nil {
      Error.Println("Error writing to InfluxDB:", err)
    }

    Info.Println("Sleeping for", POLLING_RATE)
    time.Sleep(POLLING_RATE)
  }
  Trace.Println("Stopping process thread")
}

func ParseValues(command string, values []string) *client.Point {
  Trace.Println("Parsing command:", command)
  Trace.Println("with values:")
  for _, value := range values {
    Trace.Println(value)
  }

  tags := map[string]string { }
  var fields map[string]interface{}

  if strings.Contains(command, "Temp") ||
     strings.Contains(command, "Stunden") ||
     strings.Contains(command, "Starts") {
    floatValue, err := strconv.ParseFloat(values[0], 32)
    if err != nil {
      Error.Println(err)
      return nil
    } else {
      Info.Println(command, ":", floatValue)
      fields = map[string]interface{} { strings.TrimPrefix(command, "get"): floatValue }
    }
  } else if strings.Contains(command, "Status") ||
            strings.Contains(command, "Sammel") {
    boolValue, err := strconv.ParseBool(values[0])
    if err != nil {
      Error.Println(err)
      return nil
    } else {
      Info.Println(command, ":", boolValue)
      fields = map[string]interface{} { strings.TrimPrefix(command, "get"): boolValue }
    }
  } else if strings.Contains(command, "Err:") {
    Error.Println("Error occured:", values[1])
    return nil
  } else {
    Error.Println("Could not parse values.")
    return nil
  }

  pt, err := client.NewPoint("viessmann", tags, fields, time.Now())
  if err != nil {
    Error.Println("Could not create point: ", err)
    return nil
  }
  Info.Println("Point created: ", pt.String())
  return pt
}

func setup() {
  for i := 0; i < MAX_RECONNECT; i++ {
    Info.Println("Attempt to connect", i+1, "of", MAX_RECONNECT, "tries.")
    if Connect() {
      viessmann.reader = bufio.NewReader(viessmann.connection)
      viessmann.writer = bufio.NewWriter(viessmann.connection)

      viessmann.channel = make(chan string)
      break
    }
    time.Sleep(5 * time.Second)
  }

  if !viessmann.connected {
    // If we are still not connected at this point we exit and eventually restart the container.
    Error.Println("Could not connect to vcontrold server. Exiting...")
    os.Exit(1)
  }
}

func main() {
  Init(os.Stdout, os.Stdout, os.Stderr)

  flag.Parse()
  Info.Println("======== ViessmannScraper ========")

  var commands []string
  commands = append(commands, TEMP_RUECKLAUF, TEMP_ABGAS, TEMP_SOLAR_DACH,
                    TEMP_SOLAR_WW, TEMP_SPEICHER, TEMP_SOLL_WW, TEMP_AUSSEN_GEDAEMPFT,
                    TEMP_AUSSEN_GEMISCHT, TEMP_IST_KESSEL, TEMP_SOLL_KESSEL)
  commands = append(commands, STATUS_SOLAR, STATUS_PUMPE_SPEICHERLADE, STATUS_PUMPE_HEIZKREIS_A1,
                    STATUS_PUMPE_HEIZKREIS_M2, STATUS_PUMPE_ZIRKULATION, STATUS_RELAIS_K12,
                    STATUS_PUMPE_INTERN, STATUS_FLOW_SWITCH)
  commands = append(commands, MISC_STARTS_BRENNER, MISC_LAUFZEIT_BRENNER, MISC_LAUFZEIT_BRENNER_STUFE1,
                    MISC_LAUFZEIT_BRENNER_STUFE2, MISC_SAMMELSTOERUNG)

  for {
    if !viessmann.connected {
      setup()

      // We suppose the read thread was broken so we start it again.
      go Read()
      go Process(commands)
    }
    time.Sleep(10 * time.Second)
  }
}
