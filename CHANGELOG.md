# Changelog

## [2.7.0](https://github.com/sprintertech/sprinter-signing/compare/v2.6.0...v2.7.0) (2025-10-31)


### Features

* across message handler ([#11](https://github.com/sprintertech/sprinter-signing/issues/11)) ([7e7902b](https://github.com/sprintertech/sprinter-signing/commit/7e7902b5878172b5e0c7773f765de4a8f3b1f73b))
* Add deployment with Portainer ([#74](https://github.com/sprintertech/sprinter-signing/issues/74)) ([a1ce255](https://github.com/sprintertech/sprinter-signing/commit/a1ce255fa6ba049fbed7eb6214ab867590e04723))
* allow for coordinator to be set manually ([#9](https://github.com/sprintertech/sprinter-signing/issues/9)) ([b3b05cc](https://github.com/sprintertech/sprinter-signing/commit/b3b05cc06808520ff73387e61ab6a80b65b3620f))
* bump solver config ([#57](https://github.com/sprintertech/sprinter-signing/issues/57)) ([7e6a7ce](https://github.com/sprintertech/sprinter-signing/commit/7e6a7ce09f0396a7cbf397835a8ae0982b43fe71))
* implement admin functions contract ([#16](https://github.com/sprintertech/sprinter-signing/issues/16)) ([6f4ace1](https://github.com/sprintertech/sprinter-signing/commit/6f4ace1ab9d028f2aa9c7167d736da52e0f98890))
* integrate solver config ([#41](https://github.com/sprintertech/sprinter-signing/issues/41)) ([b21c36b](https://github.com/sprintertech/sprinter-signing/commit/b21c36b9105dc84eed9c84bf31f23c31be80a45a))
* lifi escrow message handler ([#72](https://github.com/sprintertech/sprinter-signing/issues/72)) ([2fd0efd](https://github.com/sprintertech/sprinter-signing/commit/2fd0efd28ef8ca025184e8f9346e4d3153b544c3))
* lighter handler ([#89](https://github.com/sprintertech/sprinter-signing/issues/89)) ([fdd1230](https://github.com/sprintertech/sprinter-signing/commit/fdd1230d5879c2645c828582e3643c93074c9e89))
* match across zero address output tokens ([#29](https://github.com/sprintertech/sprinter-signing/issues/29)) ([8c6cd21](https://github.com/sprintertech/sprinter-signing/commit/8c6cd21833c5451e23a4f053290d1e767fbf0779))
* match contract signature ([#22](https://github.com/sprintertech/sprinter-signing/issues/22)) ([46e97f8](https://github.com/sprintertech/sprinter-signing/commit/46e97f8324378c5426f6e677a90e815873d31b50))
* mayan message handling ([#37](https://github.com/sprintertech/sprinter-signing/issues/37)) ([9289a38](https://github.com/sprintertech/sprinter-signing/commit/9289a38976afcfd4a569377dd5ec1a8f1fcbd92d))
* rhinestone message handler ([#56](https://github.com/sprintertech/sprinter-signing/issues/56)) ([6d09d0c](https://github.com/sprintertech/sprinter-signing/commit/6d09d0cfed4e175d8759391348e8861374523dd0))
* signing api ([#13](https://github.com/sprintertech/sprinter-signing/issues/13)) ([de3529b](https://github.com/sprintertech/sprinter-signing/commit/de3529bc70c36e8410aa5310f1e2fc62424985fd))
* startup configuration ([#18](https://github.com/sprintertech/sprinter-signing/issues/18)) ([4058f7d](https://github.com/sprintertech/sprinter-signing/commit/4058f7dae40a083cd20686da1df8644f883422d3))
* status api ([#15](https://github.com/sprintertech/sprinter-signing/issues/15)) ([c42f422](https://github.com/sprintertech/sprinter-signing/commit/c42f42291aea12f65feaf40d488e320085bb02d2))
* unlock API ([#73](https://github.com/sprintertech/sprinter-signing/issues/73)) ([a3bf182](https://github.com/sprintertech/sprinter-signing/commit/a3bf182560a1853532290d05c8a86f614336fd2f))
* use nonce from api ([#25](https://github.com/sprintertech/sprinter-signing/issues/25)) ([ad5538a](https://github.com/sprintertech/sprinter-signing/commit/ad5538a75c1124bd481dd8840ae0f59a54170456))
* use repayer addresses on across ([#67](https://github.com/sprintertech/sprinter-signing/issues/67)) ([636d15a](https://github.com/sprintertech/sprinter-signing/commit/636d15a0c34e3bbeb075e1cede5f4c1e4b9a62c3))
* wait for confirmations based on value ([#23](https://github.com/sprintertech/sprinter-signing/issues/23)) ([f7424d4](https://github.com/sprintertech/sprinter-signing/commit/f7424d43bce133190013d7dd16a9ab47c01caefe))


### Bug Fixes

* add lighter as supported chain id ([#91](https://github.com/sprintertech/sprinter-signing/issues/91)) ([73823d5](https://github.com/sprintertech/sprinter-signing/commit/73823d5ecde17f89402fc0d09901baf003632727))
* add listen to lifi message handler ([#81](https://github.com/sprintertech/sprinter-signing/issues/81)) ([4442996](https://github.com/sprintertech/sprinter-signing/commit/444299652b62927104bb7edc25c4578714752e69))
* add repayment chain id to api ([#68](https://github.com/sprintertech/sprinter-signing/issues/68)) ([ddf6232](https://github.com/sprintertech/sprinter-signing/commit/ddf62327333a82643ccdf2f07f8e166f43c96ac9))
* assign lighter deadline ([#96](https://github.com/sprintertech/sprinter-signing/issues/96)) ([295f195](https://github.com/sprintertech/sprinter-signing/commit/295f195ca7e29602f98a786d4e70fa6fdda67ec8))
* initialize lifi escrow mh ([#82](https://github.com/sprintertech/sprinter-signing/issues/82)) ([a904117](https://github.com/sprintertech/sprinter-signing/commit/a904117ede338985c1a2f6c23cd8b6722c2b71ad))
* invalid lifi order types ([#76](https://github.com/sprintertech/sprinter-signing/issues/76)) ([3eda013](https://github.com/sprintertech/sprinter-signing/commit/3eda0132ddb72de5926117b1bedc102d5ad9a189))
* json typo in signing api ([#69](https://github.com/sprintertech/sprinter-signing/issues/69)) ([06cba74](https://github.com/sprintertech/sprinter-signing/commit/06cba7476701aeb237a85725e378220dfec081d2))
* lifi calldata token is nil ([#86](https://github.com/sprintertech/sprinter-signing/issues/86)) ([db93c51](https://github.com/sprintertech/sprinter-signing/commit/db93c51ba68008f41cc5d289452bdbc86c9a6ca3))
* lifi solver dependecy ([#78](https://github.com/sprintertech/sprinter-signing/issues/78)) ([fbd6a5e](https://github.com/sprintertech/sprinter-signing/commit/fbd6a5e327922d07ee9d48f3c12480c1625851a2))
* lighter signing hash ([#93](https://github.com/sprintertech/sprinter-signing/issues/93)) ([269976d](https://github.com/sprintertech/sprinter-signing/commit/269976d646d8b57255dcda204c5260b36ecf7cd9))
* log unlock hash data ([#95](https://github.com/sprintertech/sprinter-signing/issues/95)) ([bc6dc88](https://github.com/sprintertech/sprinter-signing/commit/bc6dc884c4c94139063616098c251f604ad90ebb))
* match signature with on-chain one ([#26](https://github.com/sprintertech/sprinter-signing/issues/26)) ([efeceb0](https://github.com/sprintertech/sprinter-signing/commit/efeceb032201f0f53c0a9c47361ce0e2ce8b307d))
* panic on invalid event ([#70](https://github.com/sprintertech/sprinter-signing/issues/70)) ([f0bb038](https://github.com/sprintertech/sprinter-signing/commit/f0bb0381b45a9d0c0c5a56632cdc22905426c9bd))
* release and build action ([#34](https://github.com/sprintertech/sprinter-signing/issues/34)) ([f5d5f9d](https://github.com/sprintertech/sprinter-signing/commit/f5d5f9d9cb686f301c7fe6cb4958ecb6c2ea0079))
* remove 0x ([#77](https://github.com/sprintertech/sprinter-signing/issues/77)) ([d1fb4f6](https://github.com/sprintertech/sprinter-signing/commit/d1fb4f6313060a9659e6750119f4011368ac0d9c))
* remove mayan driver check ([#55](https://github.com/sprintertech/sprinter-signing/issues/55)) ([648a2e4](https://github.com/sprintertech/sprinter-signing/commit/648a2e401db908ec1b2224192cc99f85a4325ca1))
* set repayer address from solver config ([#43](https://github.com/sprintertech/sprinter-signing/issues/43)) ([160c65e](https://github.com/sprintertech/sprinter-signing/commit/160c65e4b519a3d35a15b6304fbd969ae6832d51))
* set repayment address as liquidity pool ([#28](https://github.com/sprintertech/sprinter-signing/issues/28)) ([664d249](https://github.com/sprintertech/sprinter-signing/commit/664d249fa6e83d78713ddd59326f532285b74af7))
* start pyth pricer ([#79](https://github.com/sprintertech/sprinter-signing/issues/79)) ([d7ddecb](https://github.com/sprintertech/sprinter-signing/commit/d7ddecb666ff91d60dec68aac877624742d4f124))
* store signature by deposit id ([#83](https://github.com/sprintertech/sprinter-signing/issues/83)) ([e1cf98f](https://github.com/sprintertech/sprinter-signing/commit/e1cf98f25f429bdc193b84333d12581201816cf7))
* stream reseting before closing connection ([#71](https://github.com/sprintertech/sprinter-signing/issues/71)) ([f517337](https://github.com/sprintertech/sprinter-signing/commit/f517337b3e9d42245a802e2ca18bcb1d36763b6a))
* switch borrowMany to borrow ([#84](https://github.com/sprintertech/sprinter-signing/issues/84)) ([387ded4](https://github.com/sprintertech/sprinter-signing/commit/387ded45ab7a866e4b5ea097017aa09876f6066e))
* update lifi filler abi ([#75](https://github.com/sprintertech/sprinter-signing/issues/75)) ([37d26f3](https://github.com/sprintertech/sprinter-signing/commit/37d26f39dd762d0e28954cffc143ee4ad1ed3f39))
* use deadline from sprinter-api on lighter ([#94](https://github.com/sprintertech/sprinter-signing/issues/94)) ([5cddb59](https://github.com/sprintertech/sprinter-signing/commit/5cddb597ce81aa84a4f16c3067c6d30e25e894f8))
* use deposit tx hash for lighter ([#92](https://github.com/sprintertech/sprinter-signing/issues/92)) ([a6b55ac](https://github.com/sprintertech/sprinter-signing/commit/a6b55aca6fb4f60a0c91819895f5bb046f84de1a))
* use deposit tx to fetch across deposit ([#40](https://github.com/sprintertech/sprinter-signing/issues/40)) ([f33a6c4](https://github.com/sprintertech/sprinter-signing/commit/f33a6c4293a23b724075dd0ddb35669e342d0252))
* use order hash as deposit ID ([#47](https://github.com/sprintertech/sprinter-signing/issues/47)) ([4c8762a](https://github.com/sprintertech/sprinter-signing/commit/4c8762af6093e174a7dee7e3b49b2ef7e419bf9c))
* use tx hash from lifi api ([#80](https://github.com/sprintertech/sprinter-signing/issues/80)) ([628a044](https://github.com/sprintertech/sprinter-signing/commit/628a0443e71132f2d14a73011e6b5ceb285be8ed))


### Miscellaneous

* add release please starting commit hash ([#36](https://github.com/sprintertech/sprinter-signing/issues/36)) ([35c0976](https://github.com/sprintertech/sprinter-signing/commit/35c0976e19ff1726f8e3e6c7283cf12f027642e2))
* cache go dep ([#87](https://github.com/sprintertech/sprinter-signing/issues/87)) ([24602d2](https://github.com/sprintertech/sprinter-signing/commit/24602d2d7807d2c78ab7bf03b558578004b11672))
* log lifi unlock hash ([#85](https://github.com/sprintertech/sprinter-signing/issues/85)) ([3fd08f2](https://github.com/sprintertech/sprinter-signing/commit/3fd08f29691504dee740fe354e5782c8f634d366))
* publish latest image on push to main ([#19](https://github.com/sprintertech/sprinter-signing/issues/19)) ([015b0ad](https://github.com/sprintertech/sprinter-signing/commit/015b0adabbf8e7f8ba0171c04a04c73d52a25358))
* publish tagged docker image on release ([#33](https://github.com/sprintertech/sprinter-signing/issues/33)) ([1ce2cb3](https://github.com/sprintertech/sprinter-signing/commit/1ce2cb37167117f721837f9ea4e05bb0d20d9c70))

## Changelog
