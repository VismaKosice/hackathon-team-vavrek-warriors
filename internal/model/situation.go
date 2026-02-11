package model

type Situation struct {
	Dossier *Dossier `json:"dossier"`
}

type Dossier struct {
	DossierID      string   `json:"dossier_id"`
	Status         string   `json:"status"`
	RetirementDate *string  `json:"retirement_date"`
	Persons        []Person `json:"persons"`
	Policies       []Policy `json:"policies"`
	PolicySeq      int      `json:"-"` // internal: next policy sequence number
}

type Person struct {
	PersonID  string `json:"person_id"`
	Role      string `json:"role"`
	Name      string `json:"name"`
	BirthDate string `json:"birth_date"`
}

type Policy struct {
	PolicyID            string       `json:"policy_id"`
	SchemeID            string       `json:"scheme_id"`
	EmploymentStartDate string       `json:"employment_start_date"`
	Salary              float64      `json:"salary"`
	PartTimeFactor      float64      `json:"part_time_factor"`
	AttainablePension   *float64     `json:"attainable_pension"`
	Projections         []Projection `json:"projections"`
}

type Projection struct {
	Date             string  `json:"date"`
	ProjectedPension float64 `json:"projected_pension"`
}
