package cmd

import (
    "fmt"
    "net"
    "github.com/fatih/color"
    terminal "github.com/wayneashleyberry/terminal-dimensions"
)

func logReplace(ip string, found, total, current int, progress float64) {
    green := color.New(color.FgGreen).SprintFunc()
    s := fmt.Sprintf("Escaneando: %s F:%s %d/%d %.2f%%", ip, green(found), current, total, progress)

    termWidth, _, err := terminal.Dimensions()
    if err == nil {
        w := int(termWidth) - 3
        if len(s) >= w {
            s = s[:w] + "..."
        }
    }

    fmt.Print("\r\033[2K", s, "\r")
}

func incrementIP(ip net.IP) {
    for j := len(ip) - 1; j >= 0; j-- {
        ip[j]++
        if ip[j] > 0 {
            break
        }
    }
}
