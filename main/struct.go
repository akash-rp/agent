package main

type systemstats struct {
	Cores       string `json:"cores"`
	Cpu         string `json:"cpu"`
	TotalMemory string `json:"totalMemory"`
	UsedMemory  string `json:"usedMemory"`
	TotalDisk   string `json:"totalDisk"`
	UsedDisk    string `json:"usedDisk"`
	Bandwidth   string `json:"bandwidth"`
	Os          string `json:"os"`
}

type wpadd struct {
	AppName       string `json:"appName"`
	UserName      string `json:"userName"`
	Url           string `json:"url"`
	Title         string `json:"title"`
	AdminUser     string `json:"adminUser"`
	AdminPassword string `json:"adminPassword"`
	AdminEmail    string `json:"adminEmail"`
	Routing       string `json:"routing"`
}

type db struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type wpdelete struct {
	Main struct {
		Name string `json:"name"`
		User string `json:"user"`
	} `json:"main"`
	Staging struct {
		Name string `json:"name"`
		User string `json:"user"`
	} `json:"staging"`
	IsStaging bool `json:"isStaging"`
}

type wpcert struct {
	AppName string `json:"appName"`
	Url     string `json:"url"`
	Email   string `json:"email"`
}

type errcode struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type DomainEdit struct {
	Name string `json:"name"`
	Site Site   `json:"site"`
}

type PrimaryChange struct {
	Name     string `json:"name"`
	MainUrl  string `json:"mainUrl"`
	AliasUrl string `json:"aliasUrl"`
	User     string `json:"user"`
}

type PHPChange struct {
	Name string `json:"name"`
	// Sites  []Site `json:"sites"`
	OldPHP string `json:"oldphp"`
	NewPHP string `json:"newphp"`
}

type PHP struct {
	MaxExecutionTime      string `ini:"max_execution_time"`
	MaxFileUploads        string `ini:"max_file_uploads"`
	MaxInputTime          string `ini:"max_input_time"`
	MaxInputVars          string `ini:"max_input_vars"`
	MemoryLimit           string `ini:"memory_limit"`
	PostMaxSize           string `ini:"post_max_size"`
	SessionCookieLifetime string `ini:"session.cookie_lifetime"`
	SessionGcMaxlifetime  string `ini:"session.gc_maxlifetime"`
	ShortOpenTag          string `ini:"short_open_tag"`
	UploadMaxFilesize     string `ini:"upload_max_filesize"`
}

type PHPini struct {
	PHP
}

//////////////////////////////////////////////////////////////////////////////////////////////////////

type Global struct {
	Datasize int `json:"dataSize"`
	Maxconn  int `json:"maxConnection"`
}
type Timeout struct {
	Connect int `json:"connect"`
	Client  int `json:"client"`
	Server  int `json:"server"`
}

type Default struct {
	Timeout Timeout `json:"timeout"`
}

type Site struct {
	Name          string   `json:"name"`
	User          string   `json:"user"`
	PrimaryDomain Domain   `json:"primaryDomain"`
	AliasDomain   []Domain `json:"aliasDomain"`
	Cache         string   `json:"cache"`
	LocalBackup   Backup   `json:"localBackup"`
	Type          string   `json:"type"`
}

type Config struct {
	Global  Global  `json:"global"`
	Default Default `json:"defaults"`
	Sites   []Site  `json:"sites"`
	SSL     bool    `json:"ssl"`
}

type Domain struct {
	Url      string `json:"url"`
	SSL      bool   `json:"ssl"`
	WildCard bool   `json:"wildcard"`
}

type Backup struct {
	Automatic bool            `json:"automatic"`
	Frequency string          `json:"frequency"`
	Time      BackupTime      `json:"time"`
	Retention BackupRetention `json:"retention"`
	LastRun   string          `json:"lastrun"`
	Created   bool            `json:"created"`
}

type BackupTime struct {
	Hour     string `json:"hour"`
	Minute   string `json:"minute"`
	MonthDay string `json:"monthday"`
	WeekDay  string `json:"weekday"`
}

type BackupRetention struct {
	Type string `json:"type"`
	Time int    `json:"time"`
}

type Staging struct {
	Name        string `json:"name"`
	User        string `json:"user"`
	Type        string `json:"type"`
	Url         string `json:"url"`
	LivesiteUrl string `json:"livesiteurl"`
}

type SyncChanges struct {
	Method string   `json:"method"`
	Type   []string `json:"type"`
	From   struct {
		Name string `json:"name"`
		User string `json:"user"`
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"fromSite"`
	To struct {
		Name string `json:"name"`
		User string `json:"user"`
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"toSite"`
	DbType      string   `json:"dbType"`
	AllSelected bool     `json:"allSelected"`
	Tables      []string `json:"tables"`
}
