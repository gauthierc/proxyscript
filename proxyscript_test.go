package main

import "testing"

func BenchmarkPacforIP(b *testing.B) {
	file := newCsv("./sys/test.csv", "./sys/")
	err := file.LoadCsvFile()
	if err != nil {
		b.Fatalf("Erreur %v", err)
	}
	for i := 0; i < b.N; i++ {
		filepac, _ := file.PacforIP("192.168.0.1")
		if filepac != "net1920.pac" {
			b.Fatalf("Erreur")
		}
	}
}

func TestPacforIP(t *testing.T) {
	file := newCsv("./sys/test.csv", "./sys/")
	err := file.LoadCsvFile()
	if err != nil {
		t.Fatalf("Erreur %v", err)
	}
	tt := []struct {
		remoteip   string
		fichierpac string
	}{
		{"192.168.1.45", "net1920.pac"},
		{"192.168.3.22", "net1923.pac"},
		{"192.168.4.10", "net1924.pac"},
		{"10.0.5.3", "net10.pac"},
	}
	for _, tc := range tt {
		t.Run(tc.remoteip, func(t *testing.T) {
			filepac, _ := file.PacforIP(tc.remoteip)
			if filepac != tc.fichierpac {
				t.Fatalf("Fichier retourné %s pour l'ip %s alors que %s attendu", filepac, tc.remoteip, tc.fichierpac)
			}
		})
	}
	err = file.UpdateCsvFile("./sys/error.csv")
	if err == nil {
		t.Fatalf("Erreur %v", err)
	}
	err = file.UpdateCsvFile("./sys/test.csv")
	if err != nil {
		t.Fatalf("Erreur %v", err)
	}
}
