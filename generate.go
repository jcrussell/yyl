package main

// Chart based on: https://codepen.io/Dannzzor/pen/zoJGw

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Rating struct {
	Number int
	Date   time.Time
	Value  float32
	Max    float32

	FormattedDate string
}

type MenuItem struct {
	Number int
	Name   string

	Ratings map[string]Rating
}

type Stats struct {
	HasDate      bool
	MaxPerWeek   int
	Longest      time.Duration
	LongestAfter string

	Weekdays      []int
	WeekdayRatios []float32
	Ratings       []float32
	RatingRatios  []float32

	FormattedLongest string
}

// readMenu from file, all errors are fatal
func readMenu(fname string) []MenuItem {
	var menu []MenuItem

	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)

	// ignore header
	r.Read()

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if len(record) != 2 {
			log.Fatalf("invalid record in %v", fname)
		}

		i, err := strconv.Atoi(record[0])
		if err != nil {
			log.Fatal(err)
		}

		menu = append(menu, MenuItem{
			Number:  i,
			Name:    record[1],
			Ratings: map[string]Rating{},
		})
	}

	return menu
}

// readRatings from file, all errors are fatal
func readRatings(fname string) []Rating {
	var ratings []Rating

	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)

	// ignore header
	r.Read()

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if len(record) != 4 {
			log.Fatalf("invalid record in %v", fname)
		}

		r := Rating{}

		r.Number, err = strconv.Atoi(record[0])
		if err != nil {
			log.Fatal(err)
		}

		if record[1] != "" {
			r.Date, err = time.Parse("20060102", record[1])
			if err != nil {
				log.Fatal(err)
			}
			r.FormattedDate = r.Date.Format("Mon Jan 2 2006")
		}

		tf, err := strconv.ParseFloat(record[2], 32)
		if err != nil {
			log.Fatal(err)
		}
		r.Value = float32(tf)

		tf, err = strconv.ParseFloat(record[3], 32)
		if err != nil {
			log.Fatal(err)
		}
		r.Max = float32(tf)

		ratings = append(ratings, r)
	}

	return ratings
}

func main() {
	menu := readMenu("menu.csv")

	files, err := ioutil.ReadDir("ratings")
	if err != nil {
		log.Fatal(err)
	}

	stats := map[string]Stats{}

	for _, fi := range files {
		fname := filepath.Join("ratings", fi.Name())

		who := strings.TrimSuffix(fi.Name(), ".csv")
		who = strings.Title(who)

		s := Stats{
			Weekdays: make([]int, 7),
		}

		count := 0

		week := 0
		weekcount := 1
		var prev time.Time

		for _, rating := range readRatings(fname) {
			var name string

			// attach ratings to menu items
			for i := range menu {
				if rating.Number == menu[i].Number {
					menu[i].Ratings[who] = rating
					name = menu[i].Name
					break
				}
			}

			// number of entries
			count += 1

			// compute frequency of ratings
			if s.Ratings == nil {
				// assume each has the same max
				s.Ratings = make([]float32, int(rating.Max+1))
			}

			s.Ratings[int(rating.Value)] += 1

			// some don't have dates
			if !rating.Date.IsZero() {
				// compute frequency plots for day of the week
				s.Weekdays[rating.Date.Weekday()] += 1

				if _, v := rating.Date.ISOWeek(); v == week {
					weekcount += 1
				} else {
					if weekcount > s.MaxPerWeek {
						s.MaxPerWeek = weekcount
					}
					week = v
					weekcount = 1
				}

				if prev.IsZero() {
					prev = rating.Date
				}
				if v := rating.Date.Sub(prev); v > s.Longest {
					s.Longest = v
					s.LongestAfter = name
				}

				prev = rating.Date

				s.HasDate = true
			}
		}

		s.WeekdayRatios = make([]float32, len(s.Weekdays))
		for i := 0; i < len(s.Weekdays); i++ {
			s.WeekdayRatios[i] = float32(s.Weekdays[i]) / float32(count) * 100
		}

		s.RatingRatios = make([]float32, len(s.Ratings))
		for i := 0; i < len(s.Ratings); i++ {
			s.RatingRatios[i] = float32(s.Ratings[i]) / float32(count) * 100
		}

		s.FormattedLongest = fmt.Sprintf("%.f days", s.Longest.Hours()/24)

		stats[who] = s
	}

	tmpl := template.Must(template.New("test").Parse(page))
	tmpl.Execute(os.Stdout, struct {
		Menu  []MenuItem
		Stats map[string]Stats
	}{menu, stats})
}

var page = `<html>
<head>
<style>
img {
	width: 400px;
}
div.item {
	float: left;
	padding: 10px;
}
#content {
	padding: 10px;
}
div.ratings {
	padding: 5px;
}
hr.clear, br.clear {
	clear: both;
}

.chart {
	width: 500px;
	background: #fff;
	overflow: hidden;
	float: left;
	padding: 10px;
}

.progress-bar {
	float: left;
	height: 300px;
	width: 40px;
	margin-right: 25px;
}

.progress-track {
	position: relative;
	width: 40px;
	height: 100%;
	background: #ebebeb;
}

.progress-fill {
	position: relative;
	background: #825;
	height: 50%;
	width: 40px;
	color: #fff;
	text-align: center;
	font-family: "Lato","Verdana",sans-serif;
	font-size: 12px;
	line-height: 20px;
}
</style>
<script src="https://code.jquery.com/jquery-3.2.1.min.js"></script>
<script>
$(document).ready(function() {
	$(".progress-fill span").each(function(){
		var percent = $(this).html();
		var pTop = 100 - ( percent.slice(0, percent.length - 1) ) + "%";
		$(this).parent().css({
			"height" : percent,
			"top" : pTop
		});
	});
});
</script>
</head>
<body>
<div id="content">
<h1>Year of the YYL</h1>

<p>
In 2015, three boys decided to embark on an epic challenge: eat all 40 items on
the Yin Yin menu, in order, in less than a year. Three men emerged, victorious.
</p>

<h2>Ratings</h2>

<div id="items">
{{ range .Menu }}
	<div class="item">
	<h3>#{{.Number}}: {{.Name}}</h3>
	<img src="img/{{ printf "%02d" .Number}}.jpg" title="{{.Name}}" />
	<div class="ratings">
		<ul>
		{{- range $who, $rating := .Ratings }}
			<li>
				{{- $who }}: {{ $rating.Value }}/{{ $rating.Max }}
				{{- if not $rating.Date.IsZero }} on {{ $rating.FormattedDate }}{{ end -}}
			</li>
		{{- end }}
		</ul>
	</div>
	</div>
{{ end }}
</div>

<hr class="clear" />

<h2>Statistics</h2>

{{ range $who, $stats := .Stats }}
	<h3>{{ $who }}</h3>

	{{ if .HasDate }}
		<p>Most visits in a week: {{ .MaxPerWeek }}</p>
		<p>Longest time between YYLs: {{ .FormattedLongest }} after {{ .LongestAfter }}</p>

		<div class="chart">
		<h4>Day of Week</h4>
		{{ range $k, $v := .WeekdayRatios }}
			<div class="progress-bar">
				<div class="progress-track">
					<div class="progress-fill">
						<span>{{ printf "%2.f" $v }}%</span>
					</div>
				</div>
			</div>
		{{ end }}
		</div>
	{{ end }}

	<div class="chart">
	<h4>Rating</h4>
	{{ range $k, $v := .RatingRatios }}
		<div class="progress-bar">
			<div class="progress-track">
				<div class="progress-fill">
					<span>{{ printf "%2.f" $v }}%</span>
				</div>
			</div>
		</div>
	{{ end }}
	</div>

	<br class="clear" />
{{ end }}

</div>
</body>
</html>`
