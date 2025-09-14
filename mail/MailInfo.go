package main

type MailAppInfo struct {
	ApplicationName string `json:"application_name"`
	SystemName      string `json:"system_name"`
	GroupName       string `json:"group_name"`
	Recommend       string `json:"recommend"`
}

type CpuLimitWeekData struct {
	GroupName    string  `json:"group_name"`
	ContainerAvg float64 `json:"container_avg"`
	PodAvg       float64 `json:"pod_avg"`
}

type MemLimitWeekData struct {
	GroupName    string  `json:"group_name"`
	ContainerAvg float64 `json:"container_avg"`
	PodAvg       float64 `json:"pod_avg"`
}

type CpuCoreWeekData struct {
	GroupName    string  `json:"group_name"`
	ContainerAvg float64 `json:"container_avg"`
	PodAvg       float64 `json:"pod_avg"`
}

type MemAvgWeekData struct {
	GroupName    string  `json:"group_name"`
	ContainerAvg float64 `json:"container_avg"`
	PodAvg       float64 `json:"pod_avg"`
}

type MailTotalDataInfo struct {
	StartTime     string
	EndTime       string
	OverWeekData  []OverWeekData
	BelowWeekData []BelowWeekData
}

type OverWeekData struct {
	ApplicationName   string  `json:"application_name"`
	SystemName        string  `json:"system_name"`
	GroupName         string  `json:"group_name"`
	ContainerCpuAvg   float64 `json:"container_cpu_avg"`
	PodCpuAvg         float64 `json:"pod_cpu_avg"`
	ContainerMemAvg   float64 `json:"container_mem_avg"`
	PodMemAvg         float64 `json:"pod_mem_avg"`
	ContainerCoreAvg  float64 `json:"container_core_avg"`
	PodCoreAvg        float64 `json:"pod_core_avg"`
	ContainerMemValue float64 `json:"container_mem_value"`
	PodMemValue       float64 `json:"pod_mem_value"`
	Recommend         string  `json:"recommend"`
}

type BelowWeekData struct {
	ApplicationName   string  `json:"application_name"`
	SystemName        string  `json:"system_name"`
	GroupName         string  `json:"group_name"`
	ContainerCpuAvg   float64 `json:"container_cpu_avg"`
	PodCpuAvg         float64 `json:"pod_cpu_avg"`
	ContainerMemAvg   float64 `json:"container_mem_avg"`
	PodMemAvg         float64 `json:"pod_mem_avg"`
	ContainerCoreAvg  float64 `json:"container_core_avg"`
	PodCoreAvg        float64 `json:"pod_core_avg"`
	ContainerMemValue float64 `json:"container_mem_value"`
	PodMemValue       float64 `json:"pod_mem_value"`
	Recommend         string  `json:"recommend"`
}

type MailExcelDataInfo struct {
	OverWeekExcelData  []OverWeekExcelData
	BelowWeekExcelData []BelowWeekExcelData
}

type OverWeekExcelData struct {
	ApplicationName   string  `json:"application_name"`
	SystemName        string  `json:"system_name"`
	GroupName         string  `json:"group_name"`
	ContainerCpuAvg   float64 `json:"container_cpu_avg"`
	PodCpuAvg         float64 `json:"pod_cpu_avg"`
	ContainerMemAvg   float64 `json:"container_mem_avg"`
	PodMemAvg         float64 `json:"pod_mem_avg"`
	ContainerCoreAvg  float64 `json:"container_core_avg"`
	PodCoreAvg        float64 `json:"pod_core_avg"`
	ContainerMemValue float64 `json:"container_mem_value"`
	PodMemValue       float64 `json:"pod_mem_value"`
	Recommend         string  `json:"recommend"`
}

type BelowWeekExcelData struct {
	ApplicationName   string  `json:"application_name"`
	SystemName        string  `json:"system_name"`
	GroupName         string  `json:"group_name"`
	PodId             string  `json:"pod_id"`
	ContainerCpuAvg   float64 `json:"container_cpu_avg"`
	PodCpuAvg         float64 `json:"pod_cpu_avg"`
	ContainerMemAvg   float64 `json:"container_mem_avg"`
	PodMemAvg         float64 `json:"pod_mem_avg"`
	ContainerCoreAvg  float64 `json:"container_core_avg"`
	PodCoreAvg        float64 `json:"pod_core_avg"`
	ContainerMemValue float64 `json:"container_mem_value"`
	PodMemValue       float64 `json:"pod_mem_value"`
	Recommend         string  `json:"recommend"`
}
