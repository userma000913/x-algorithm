use crate::schema::{events::Event, tweet_events::TweetEvent};
use anyhow::{Context, Result};
use prost::Message;
use thrift::protocol::{TBinaryInputProtocol, TSerializable};
use xai_thunder_proto::InNetworkEvent;

/// 将 Thrift 二进制消息反序列化为 TweetEvent
pub fn deserialize_tweet_event(payload: &[u8]) -> Result<TweetEvent> {
    let mut cursor = std::io::Cursor::new(payload);
    let mut protocol = TBinaryInputProtocol::new(&mut cursor, true);

    TweetEvent::read_from_in_protocol(&mut protocol).context("Failed to deserialize TweetEvent")
}

/// 将 Thrift 二进制消息反序列化为 Event
pub fn deserialize_event(payload: &[u8]) -> Result<Event> {
    let mut cursor = std::io::Cursor::new(payload);
    let mut protocol = TBinaryInputProtocol::new(&mut cursor, true);

    Event::read_from_in_protocol(&mut protocol).context("Failed to deserialize Event")
}

/// 将 proto 二进制消息反序列化为 InNetworkEvent
pub fn deserialize_tweet_event_v2(payload: &[u8]) -> Result<InNetworkEvent> {
    InNetworkEvent::decode(payload).context("Failed to deserialize InNetworkEvent")
}
