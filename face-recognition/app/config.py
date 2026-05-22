import os
from typing import List
import yaml
from dataclasses import dataclass, field


@dataclass
class KafkaConfig:
    brokers: List[str] = field(default_factory=lambda: ["localhost:9092"])
    frame_topic: str = "t_dorm_frame"
    event_topic: str = "t_dorm_event"
    group_id: str = "face-recognition-group"
    max_poll_records: int = 10


@dataclass
class DetectionConfig:
    model_path: str = "app/models/detection.onnx"
    confidence_threshold: float = 0.6
    input_size: List[int] = field(default_factory=lambda: [640, 640])
    min_face_size: int = 80
    blur_threshold: float = 100.0
    nms_iou_threshold: float = 0.5


@dataclass
class FeatureConfig:
    model_path: str = "app/models/feature.onnx"
    embedding_size: int = 512


@dataclass
class MatchConfig:
    method: str = "sims_api"
    sims_api_url: str = ""
    sims_api_timeout: float = 3.0
    auth_token: str = ""
    cache_ttl: int = 3600
    match_threshold: float = 0.65
    fallback_to_cache: bool = True

    def get_auth_token(self) -> str:
        return os.getenv("FACE_AUTH_TOKEN", self.auth_token)


@dataclass
class DirectionConfig:
    method: str = "roi_line"
    roi_line_x: float = 0.5
    min_track_points: int = 3


@dataclass
class DedupConfig:
    window_seconds: int = 10
    max_cache_size: int = 1000


@dataclass
class StrangerConfig:
    enabled: bool = True
    alert_threshold: float = 0.45


@dataclass
class NightModeConfig:
    enabled: bool = True
    start_hour: int = 22
    end_hour: int = 6
    clahe_clip_limit: float = 2.0


@dataclass
class RedisConfig:
    host: str = "localhost"
    port: int = 6379
    db: int = 0
    socket_timeout: float = 2.0


@dataclass
class LogConfig:
    level: str = "INFO"


@dataclass
class BehaviorEventConfig:
    enabled: bool = True
    event_type_mapping: dict = field(default_factory=lambda: {
        "loitering": "loiter",
        "running": "running",
        "zone_intrusion": "zone",
        "crowd_alert": "crowd",
    })


@dataclass
class BehaviorConfig:
    enabled: bool = False
    loitering_threshold_seconds: float = 30.0
    loitering_radius_px: float = 50.0
    running_speed_threshold_px_per_sec: float = 200.0
    crowd_threshold_count: int = 5
    crowd_debounce_frames: int = 3
    zones: list = field(default_factory=list)
    event_cooldown_seconds: float = 30.0
    event: BehaviorEventConfig = field(default_factory=BehaviorEventConfig)


@dataclass
class AppConfig:
    kafka: KafkaConfig = field(default_factory=KafkaConfig)
    detection: DetectionConfig = field(default_factory=DetectionConfig)
    feature: FeatureConfig = field(default_factory=FeatureConfig)
    match: MatchConfig = field(default_factory=MatchConfig)
    direction: DirectionConfig = field(default_factory=DirectionConfig)
    dedup: DedupConfig = field(default_factory=DedupConfig)
    stranger: StrangerConfig = field(default_factory=StrangerConfig)
    night_mode: NightModeConfig = field(default_factory=NightModeConfig)
    redis: RedisConfig = field(default_factory=RedisConfig)
    log: LogConfig = field(default_factory=LogConfig)
    behavior: BehaviorConfig = field(default_factory=BehaviorConfig)


def load_config(path: str = "config.yaml") -> AppConfig:
    with open(path) as f:
        data = yaml.safe_load(f)

    cfg = AppConfig()

    if "kafka" in data:
        cfg.kafka = KafkaConfig(**data["kafka"])
    if "detection" in data:
        cfg.detection = DetectionConfig(**data["detection"])
    if "feature" in data:
        cfg.feature = FeatureConfig(**data["feature"])
    if "match" in data:
        cfg.match = MatchConfig(**data["match"])
    if "direction" in data:
        cfg.direction = DirectionConfig(**data["direction"])
    if "dedup" in data:
        cfg.dedup = DedupConfig(**data["dedup"])
    if "stranger" in data:
        cfg.stranger = StrangerConfig(**data["stranger"])
    if "night_mode" in data:
        cfg.night_mode = NightModeConfig(**data["night_mode"])
    if "redis" in data:
        cfg.redis = RedisConfig(**data["redis"])
    if "log" in data:
        cfg.log = LogConfig(**data["log"])
    if "behavior" in data:
        bdata = data["behavior"]
        if "event" in bdata:
            bdata["event"] = BehaviorEventConfig(**bdata["event"])
        cfg.behavior = BehaviorConfig(**bdata)

    return cfg
