package messages

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type MessageTable struct {
	table *strings.Builder
	cols  int
}

const (
	MAX_MESSAGE_LENGTH_WITH_IMAGE    = 131072
	MAX_MESSAGE_LENGTH_WITHOUT_IMAGE = 5000
)

// MakeTable generates a html table with the first parameter as a header on top of the table
// and subsequents as column headers for the table
func MakeTable(header string, columns ...string) MessageTable {
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
	return MessageTable{&b, len(columns)}
}

// AddRow adds cells to a MessageTable
func (msgTbl MessageTable) AddRow(cells ...string) {
	fmt.Fprintf(msgTbl.table, "<tr>")
	for _, v := range cells {
		fmt.Fprintf(msgTbl.table, "<td><p>%s</p></td>", v)
	}
	fmt.Fprintf(msgTbl.table, "</tr>")
}

// String escapes the tbody and table elements of a MessageTable and then returns a string of the MessageTable
func (msgTbl MessageTable) String() string {
	fmt.Fprintf(msgTbl.table, "</tbody></table>")
	return msgTbl.table.String()
}

func FindCoverArtPath(playPath string) string {
	basedir := filepath.Dir(playPath)
	// Todo, robust matching
	if _, err := os.Stat(basedir + "/cover.jpg"); err == nil {
		return basedir + "/cover.jpg"
	}
	if _, err := os.Stat(basedir + "/cover.png"); err == nil {
		return basedir + "/cover.png"
	}
	return ""
}

// generateCoverArtImage creates a base64 encoded html <img>
// TODO: Find a way to get generated cover art to follow the larger limits (for messages that contain images)
//       for now, we make sure the image is less than maxSize to be well below the 5000 text limit the mumble server
//       imposes by default for text messages (that contain no image)
func GenerateCoverArtImg(filepath string) string {
	img, err := decodeImage(filepath)
	if err != nil {
		log.Println("Failed to decode img: ", filepath, " ", err)
		return ""
	}
	resizedImg := resize.Resize(100, 100, img, resize.Lanczos3)
	jpegQuality := 60
	maxSize := 3000
	var buf bytes.Buffer
	for maxSize >= 3000 && jpegQuality > 0 {
		buf.Reset()
		options := jpeg.Options{Quality: jpegQuality}
		if err := jpeg.Encode(&buf, resizedImg, &options); err != nil {
			log.Println("Error encoding jpg for base64: ", filepath, " ", err)
			return ""
		}
		maxSize = len(buf.Bytes())
		jpegQuality -= 5 // Lower potential future jpegQuality if there's further passes
	}

	// Size Note: Grows ~1k bytes after encoding
	encodedStr := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "<img src=\"data:img/jpeg;base64, " + encodedStr + "\" />"
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
