package commands

import (
	"fmt"
	"strings"
)

type messageTable struct {
    table *strings.Builder
    cols int
}

// makeTable generates an html table with the first parameter as a header on top of the table
// and subsequents as column headers for the table
func makeTable(header string, columns ...string) messageTable {
    var b strings.Builder
    fmt.Fprintf(&b, "<h2 style=\"margin-top:16px; margin-bottom:2px; margin-left:0px; margin-right:0px; -qt-block-indent:0; text-indent:0px;\"><b><u><span style=\"font-size:x-large\">%s</span></u></b></h2>", header)
    fmt.Fprintf(&b, "<table align=\"left\" border=\"0\" style=\"margin-top:0px; margin-bottom:0px; margin-left:0px; margin-right:0px;\" cellspacing=\"2\" cellpadding=\"0\"><thead>")
    if len(columns) != 0 {
        fmt.Fprintf(&b, "<tr>")
    }
    for _, v := range columns {
        fmt.Fprintf(&b, "<th align=\"left\">%s</th>", v)
    }
    if len(columns) != 0 {
        fmt.Fprintf(&b, "</tr>")
    }
    fmt.Fprintf(&b, "</thead><tbody>")
    return messageTable{&b, len(columns)}
}

// addRow adds cells to a messageTable
func (msgTbl messageTable) addRow(cells ...string) {
    fmt.Fprintf(msgTbl.table, "<tr>")
    for _,v := range cells {
        fmt.Fprintf(msgTbl.table, "<td><p>%s</p></td>", v)
    }
    fmt.Fprintf(msgTbl.table, "</tr>")
}

// String escapes the tbody and table elements of a messageTable and then returns a string of the messageTable
func (msgTbl messageTable) String() string {
    fmt.Fprintf(msgTbl.table, "</tbody></table>")
    return msgTbl.table.String()
}