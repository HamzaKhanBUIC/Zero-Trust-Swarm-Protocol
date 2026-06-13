from setuptools import setup, find_packages

setup(
    name="swarm-mtls",
    version="1.0.0",
    description="Python SDK for the Zero-Trust Swarm Protocol",
    author="Hamza Imran",
    packages=find_packages(),
    install_requires=[
        "requests>=2.25.1",
        "openai>=1.0.0"
    ],
    python_requires=">=3.8",
)
