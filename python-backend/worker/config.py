import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    redis_url: str
    api_base_url: str
    internal_api_token: str
    s3_endpoint_url: str
    s3_access_key: str
    s3_secret_key: str
    s3_bucket: str
    s3_region: str
    blpop_timeout_seconds: int


def load_config() -> Config:
    return Config(
        redis_url=os.environ.get("REDIS_URL", "redis://localhost:6379/0"),
        api_base_url=os.environ.get("API_BASE_URL", "http://localhost:8080").rstrip("/"),
        internal_api_token=os.environ["INTERNAL_API_TOKEN"],
        s3_endpoint_url=os.environ.get("S3_ENDPOINT_URL", "http://localhost:9000"),
        s3_access_key=os.environ.get("S3_ACCESS_KEY", "minioadmin"),
        s3_secret_key=os.environ.get("S3_SECRET_KEY", "minioadmin"),
        s3_bucket=os.environ.get("S3_BUCKET", "gradle-artifacts"),
        s3_region=os.environ.get("S3_REGION", "us-east-1"),
        blpop_timeout_seconds=int(os.environ.get("BLPOP_TIMEOUT_SECONDS", "5")),
    )
