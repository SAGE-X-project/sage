from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="sage-contracts",
    version="1.0.0",
    author="SAGE Team",
    description="Python bindings for SAGE smart contracts",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/sage-x-project/sage",
    packages=find_packages(),
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.8",
    install_requires=[
        "web3>=6.11.0",
        "eth-account>=0.10.0",
        "hexbytes>=0.3.0",
    ],
)
