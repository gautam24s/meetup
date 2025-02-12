package meetup

func DefaultQualityLevels() []QualityLevel {
	return []QualityLevel{
		QualityHigh,
		QualityMid,
		QualityLow,
		QualityLowMid,
		QualityLowLow,
	}
}
