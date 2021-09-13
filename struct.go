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
	AppName       string   `json:"appName"`
	UserName      string   `json:"userName"`
	Url           string   `json:"url"`
	Title         string   `json:"title"`
	AdminUser     string   `json:"adminUser"`
	AdminPassword string   `json:"adminPassword"`
	AdminEmail    string   `json:"adminEmail"`
	SubDomain     bool     `json:"subdomain"`
	Routing       string   `json:"routing"`
	Sites         []Site   `json:"sites"`
	Exclude       []string `json:"exclude"`
}

type db struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type wpdelete struct {
	AppName  string `json:"appName"`
	UserName string `json:"userName"`
	DbName   string `json:"dbName"`
	DbUser   string `json:"DbUser"`
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
	Name  string `json:"name"`
	Sites []Site `json:"sites"`
}

type PrimaryChange struct {
	Name     string `json:"name"`
	Sites    []Site `json:"sites"`
	MainUrl  string `json:"mainUrl"`
	AliasUrl string `json:"aliasUrl"`
	User     string `json:"user"`
}

type PHPChange struct {
	Name   string `json:"name"`
	Sites  []Site `json:"sites"`
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
	Exclude       []string `json:"exclude"`
	LocalBackup   Backup   `json:"localBackup"`
}

type Config struct {
	Global  Global  `json:"global"`
	Default Default `json:"defaults"`
	Sites   []Site  `json:"sites"`
	SSL     bool    `json:"ssl"`
}

type Domain struct {
	Url       string `json:"url"`
	SubDomain bool   `json:"subDomain"`
	SSL       bool   `json:"ssl"`
	WildCard  bool   `json:"wildcard"`
	Routing   string `json:"routing"`
}

type Backup struct {
	Frequency string `json:"frequency"`
	Minute    int    `json:"minute"`
	Time      string `json:"time"`
	MonthDay  int    `json:"monthday"`
	WeekDay   string `json:"weekday"`
}
