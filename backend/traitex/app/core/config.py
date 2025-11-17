from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Minimal configuration"""
    BATCH_SIZE: int = 10

    class Config:
        env_file = ".env"


settings = Settings()
