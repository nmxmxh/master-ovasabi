# Package broadcast

## Types

### AssetBroadcaster

#### Methods

##### Publish

##### Subscribe

##### Unsubscribe

### ServiceImpl

ServiceImpl implements the BroadcastService interface.

#### Methods

##### BroadcastAction

BroadcastAction implements the BroadcastAction RPC method.

##### GetBroadcast

GetBroadcast retrieves a specific broadcast by ID.

##### ListBroadcasts

ListBroadcasts retrieves a list of broadcasts with pagination.

##### PublishLiveAssetChunk

PublishLiveAssetChunk pushes a live asset chunk to all subscribers

##### SubscribeToActions

SubscribeToActions implements the SubscribeToActions streaming RPC method.

##### SubscribeToLiveAssetChunks

SubscribeToLiveAssetChunks streams live asset chunks to the client
