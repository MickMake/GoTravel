package exporters

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/MickMake/GoTravel/storage"
)

type CSV struct{}

func (CSV) Export(w io.Writer, points []storage.Point) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"dt", "lat", "lng", "altitude", "angle", "speed", "params"}); err != nil {
		return err
	}
	for _, p := range points {
		row := []string{
			p.DT.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.7f", p.Lat),
			fmt.Sprintf("%.7f", p.Lng),
			fmt.Sprintf("%.0f", p.Altitude),
			fmt.Sprintf("%.0f", p.Angle),
			fmt.Sprintf("%.0f", p.Speed),
			p.Params,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
