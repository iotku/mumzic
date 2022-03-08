package commands

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/iotku/mumzic/playback"
	"github.com/nfnt/resize"
	"image"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type messageTable struct {
	table *strings.Builder
	cols  int
}

// makeTable generates a html table with the first parameter as a header on top of the table
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
	for _, v := range cells {
		fmt.Fprintf(msgTbl.table, "<td><p>%s</p></td>", v)
	}
	fmt.Fprintf(msgTbl.table, "</tr>")
}

// String escapes the tbody and table elements of a messageTable and then returns a string of the messageTable
func (msgTbl messageTable) String() string {
	fmt.Fprintf(msgTbl.table, "</tbody></table>")
	return msgTbl.table.String()
}

func FindCoverArtPath(player *playback.Player) string {
	basedir := filepath.Dir(player.Playlist.GetCurrentPath())
	// Todo, robust matching
	if _, err := os.Stat(basedir + "/cover.jpg"); err == nil {
		return basedir + "cover.jpg"
	}
	if _, err := os.Stat(basedir + "/cover.png"); err == nil {
		return basedir + "cover.png"
	}
	return ""
}

// generateCoverArtImage creates a base64 encoded html <img>
// TODO: Check for filesize limits for mumble servers
func GenerateCoverArtImg(filepath string) string {
	img, err := decodeImage(filepath)
	if err != nil {
		log.Println("Failed to decode img: ", filepath, " ", err)
		return ""
	}
	resizedImg := resize.Resize(200, 200, img, resize.Lanczos3)
	var buf bytes.Buffer
	if err := png.Encode(&buf, resizedImg); err != nil {
		log.Println("Error encoding png for base64: ", filepath, " ", err)
		return ""
	}

	encodedStr := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "<img src=\"data:img/png;base64, " + encodedStr + "\" />"
}

func decodeImage(filepath string) (image.Image, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	return img, nil
}
