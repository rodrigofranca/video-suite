"""S3-compatible storage. It mirrors packages/media-core/storage (Go): path-style
addressing and checksums only when the API requires them."""

from __future__ import annotations

import boto3
from botocore.config import Config as BotoConfig
from botocore.exceptions import ClientError

from .config import Config


class Storage:
    def __init__(self, cfg: Config):
        self.bucket = cfg.s3_bucket
        self.s3 = boto3.client(
            "s3",
            endpoint_url=cfg.s3_endpoint,
            region_name=cfg.s3_region,
            aws_access_key_id=cfg.s3_access_key,
            aws_secret_access_key=cfg.s3_secret_key,
            config=BotoConfig(
                s3={"addressing_style": "path"},
                request_checksum_calculation="when_required",
                response_checksum_validation="when_required",
                retries={"max_attempts": 3},
            ),
        )

    def check(self) -> None:
        """Fail when the bucket is unreachable or the credentials are wrong."""
        self.s3.head_bucket(Bucket=self.bucket)

    def exists(self, key: str) -> bool:
        try:
            self.s3.head_object(Bucket=self.bucket, Key=key)
            return True
        except ClientError as e:
            if e.response.get("Error", {}).get("Code") in ("404", "NoSuchKey", "NotFound"):
                return False
            raise

    def download_file(self, key: str, path: str) -> None:
        self.s3.download_file(self.bucket, key, path)

    def upload_file(self, key: str, path: str, content_type: str) -> None:
        self.s3.upload_file(path, self.bucket, key, ExtraArgs={"ContentType": content_type})

    def upload_text(self, key: str, text: str, content_type: str) -> None:
        self.s3.put_object(Bucket=self.bucket, Key=key, Body=text.encode("utf-8"), ContentType=content_type)

    def delete(self, key: str) -> None:
        self.s3.delete_object(Bucket=self.bucket, Key=key)

    def presign_get(self, key: str, expires: int = 3600) -> str:
        """A temporary link to read an object, for tools that stream it (a
        secret: never log it)."""
        return self.s3.generate_presigned_url(
            "get_object", Params={"Bucket": self.bucket, "Key": key}, ExpiresIn=expires
        )
