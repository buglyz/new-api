package model

func ListChannelMonitorChannels() ([]*Channel, error) {
	var channels []*Channel
	err := DB.Model(&Channel{}).
		Select("id, type, status, models, model_mapping, settings").
		Find(&channels).Error
	return channels, err
}
