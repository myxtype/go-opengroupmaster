package service

import "fmt"

func (s *Service) NewbieLimitViewByTGGroupID(tgGroupID int64) (*NewbieLimitView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureNewbieLimit, false)
	if err != nil {
		return nil, err
	}
	minutes, err := s.getNewbieLimitMinutes(group.ID)
	if err != nil {
		return nil, err
	}
	return &NewbieLimitView{
		Enabled: enabled,
		Minutes: minutes,
	}, nil
}

func (s *Service) SetNewbieLimitEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureNewbieLimit, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_newbie_limit_enabled_%t", enabled), 0, 0)
	return enabled, nil
}
