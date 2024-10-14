package cmd

import (
    "bufio"
    "context"
    "crypto/tls"
    "fmt"
    "net"
    "os"
    "strings"
    "time"

    "github.com/spf13/cobra"

    "github.com/Pablo0303/Alama/pkg/queuescanner"
)

var sniCmd = &cobra.Command{
    Use:   "sni",
    Short: "Scan server name indication list from file",
    Run:   runScanSNI,
}

var (
    sniFlagFilename string
    sniFlagDeep     int
    sniFlagTimeout  int
    sniFlagDelay    int    // Nuevo campo para el delay
    sniFlagProxy    string // Nuevo campo para el proxy
)

func init() {
    scanCmd.AddCommand(sniCmd)

    sniCmd.Flags().StringVarP(&sniFlagFilename, "filename", "f", "", "domain list filename")
    sniCmd.Flags().IntVarP(&sniFlagDeep, "deep", "d", 0, "deep subdomain")
    sniCmd.Flags().IntVar(&sniFlagTimeout, "timeout", 3, "handshake timeout")
    sniCmd.Flags().IntVarP(&sniFlagDelay, "delay", "D", 0, "delay between scans in milliseconds") // Cambiado a -D
    sniCmd.Flags().StringVar(&sniFlagProxy, "proxy", "", "proxy and port to use") // Mantenido proxy

    sniCmd.MarkFlagFilename("filename")
    sniCmd.MarkFlagRequired("filename")
}

func scanSNI(c *queuescanner.Ctx, p *queuescanner.QueueScannerScanParams) {
    domain := p.Data.(string)

    var conn net.Conn
    var err error

    dialCount := 0
    dialer := &net.Dialer{Timeout: 3 * time.Second} // Configura el tiempo de espera en el Dialer
    for {
        dialCount++
        if dialCount > 3 {
            return
        }
        conn, err = dialer.Dial("tcp", domain+":443") // Usa el dominio como dirección
        if err != nil {
            if e, ok := err.(net.Error); ok && e.Timeout() {
                c.LogReplace(p.Name, "-", "Dial Timeout")
                continue
            }
            c.Logf("Dial error: %s", err.Error())
            return
        }
        defer conn.Close()
        break
    }

    tlsConn := tls.Client(conn, &tls.Config{
        ServerName:         domain,
        InsecureSkipVerify: true,
    })
    defer tlsConn.Close()

    ctxHandshake, ctxHandshakeCancel := context.WithTimeout(context.Background(), time.Duration(sniFlagTimeout)*time.Second)
    defer ctxHandshakeCancel()
    err = tlsConn.HandshakeContext(ctxHandshake)
    if err != nil {
        c.ScanFailed(domain, nil)
        return
    }
    c.ScanSuccess(domain, func() {
        c.Log(colorG1.Sprint(domain))
    })

    // Delay entre escaneos
    if sniFlagDelay > 0 {
        time.Sleep(time.Duration(sniFlagDelay) * time.Millisecond)
    }
}

func runScanSNI(cmd *cobra.Command, args []string) {
    domainListFile, err := os.Open(sniFlagFilename)
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }
    defer domainListFile.Close()

    queueScanner := queuescanner.NewQueueScanner(scanFlagThreads, scanSNI) // Definido aquí

    scanner := bufio.NewScanner(domainListFile)
    for scanner.Scan() {
        line := scanner.Text()
        // Verifica si la línea contiene un rango de IP
        if strings.Contains(line, "-") {
            ips := strings.Split(line, "-")
            if len(ips) != 2 {
                fmt.Printf("Invalid IP range: %s\n", line)
                continue
            }
            startIP := net.ParseIP(strings.TrimSpace(ips[0]))
            endIP := net.ParseIP(strings.TrimSpace(ips[1]))
            if startIP == nil || endIP == nil {
                fmt.Printf("Invalid IPs: %s\n", line)
                continue
            }

            for ip := startIP; !ip.Equal(endIP); ip = incrementIP(ip) { // Captura el valor retornado
                queueScanner.Add(&queuescanner.QueueScannerScanParams{
                    Name: ip.String(),
                    Data: ip.String(),
                })
            }
            // Agrega la IP final
            queueScanner.Add(&queuescanner.QueueScannerScanParams{
                Name: endIP.String(),
                Data: endIP.String(),
            })
        } else {
            // Procesa una sola IP o dominio
            domain := line
            if sniFlagDeep > 0 {
                domainSplit := strings.Split(domain, ".")
                if len(domainSplit) >= sniFlagDeep {
                    domain = strings.Join(domainSplit[len(domainSplit)-sniFlagDeep:], ".")
                }
            }
            queueScanner.Add(&queuescanner.QueueScannerScanParams{
                Name: domain,
                Data: domain,
            })
        }
    }

    queueScanner.Start(nil)
}
