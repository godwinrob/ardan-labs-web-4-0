package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
)

var service string

func init() {
	flag.StringVar(&service, "service", "", "service to filter on")
}

func main() {
	flag.Parse()
	var b strings.Builder

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		s := scanner.Text()

		// Convert JSON to a map
		m := make(map[string]interface{})
		err := json.Unmarshal([]byte(s), &m)
		if err != nil {
			if service == "" {
				fmt.Println(s)
			}
			continue
		}

		// If a service filter was provided, check it
		if service != "" && m["service"] != service {
			continue
		}

		traceID := uuid.New().String()
		if v, ok := m["traceid"]; ok {
			traceID = fmt.Sprintf("%v", v)
		}

		b.Reset()
		b.WriteString(fmt.Sprintf("%s: %s: %s: %s: %s: %s: ",
			m["service"],
			m["ts"],
			m["level"],
			traceID,
			m["caller"],
			m["msg"],
		))

		for k, v := range m {
			switch k {
			case "service", "ts", "level", "traceid", "caller", "msg":
				continue
			}

			b.WriteString(fmt.Sprintf("%s=%v", k, v))
		}

		out := b.String()
		log.Println(out[:len(out)-2])
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}
