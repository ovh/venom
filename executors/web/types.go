package web

// Action represents what can be done with web executor
type Action struct {
	Click    *Click    `yaml:"click,omitempty"`
	Fill     []Fill    `yaml:"fill,omitempty"`
	Find     string    `yaml:"find,omitempty"`
	Navigate *Navigate `yaml:"navigate,omitempty"`
	Wait     int64     `yaml:"wait,omitempty"`
	SelectFrame *SelectFrame `yaml:"selectFrame,omitempty"`
	SelectRootFrame bool `yaml:"selectRootFrame,omitempty"`
	NextWindow bool	   `yaml:"nextWindow,omitempty"`
}

// Fill represents informations needed to fill input/textarea
type Fill struct {
	Find string  `yaml:"find,omitempty"`
	Text string  `yaml:"text,omitempty"`
	Key  *string `yaml:"key,omitempty"`
}

type Click struct {
	Find string `yaml:"find,omitempty"`
	Wait int64  `yaml:"wait"`
}

type Navigate struct {
	Url   string `yaml:"url,omitempty"`
	Reset bool   `yaml:"reset,omitempty"`
}

// SelectFrame represents informations needed to select the frame
type SelectFrame struct {
	Find string  `yaml:"find,omitempty"`
}