package model

// ChannelsExist checks channel identities without loading credential-bearing
// channel records into memory.
func ChannelsExist(channelIDs []int) (bool, error) {
	if len(channelIDs) == 0 {
		return false, nil
	}
	var count int64
	if err := DB.Model(&Channel{}).Where("id IN ?", channelIDs).Count(&count).Error; err != nil {
		return false, err
	}
	return count == int64(len(channelIDs)), nil
}
