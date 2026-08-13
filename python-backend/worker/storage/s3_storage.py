import boto3
from botocore.client import Config as BotoConfig


class S3Storage:
    def __init__(self, endpoint_url: str, access_key: str, secret_key: str, bucket: str, region: str = "us-east-1"):
        self._bucket = bucket
        self._client = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
            region_name=region,
            config=BotoConfig(s3={"addressing_style": "path"}),
        )

    def upload_bytes(self, key: str, data: bytes, content_type: str) -> None:
        self._client.put_object(Bucket=self._bucket, Key=key, Body=data, ContentType=content_type)

    def download_bytes(self, key: str) -> bytes:
        resp = self._client.get_object(Bucket=self._bucket, Key=key)
        return resp["Body"].read()
