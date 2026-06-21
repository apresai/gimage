package generate

import "testing"

func TestVertexGenerateContentURL(t *testing.T) {
	tests := []struct {
		name     string
		location string
		project  string
		model    string
		want     string
	}{
		{
			name:     "global uses the un-prefixed host",
			location: "global",
			project:  "my-proj",
			model:    "gemini-3.1-flash-image",
			want:     "https://aiplatform.googleapis.com/v1/projects/my-proj/locations/global/publishers/google/models/gemini-3.1-flash-image:generateContent",
		},
		{
			name:     "regional uses the location-prefixed host",
			location: "us-central1",
			project:  "my-proj",
			model:    "gemini-3.1-flash-image",
			want:     "https://us-central1-aiplatform.googleapis.com/v1/projects/my-proj/locations/us-central1/publishers/google/models/gemini-3.1-flash-image:generateContent",
		},
		{
			name:     "another region",
			location: "us-east5",
			project:  "p",
			model:    "m",
			want:     "https://us-east5-aiplatform.googleapis.com/v1/projects/p/locations/us-east5/publishers/google/models/m:generateContent",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := vertexGenerateContentURL(tc.location, tc.project, tc.model); got != tc.want {
				t.Errorf("vertexGenerateContentURL(%q, %q, %q) = %q, want %q", tc.location, tc.project, tc.model, got, tc.want)
			}
		})
	}
}
