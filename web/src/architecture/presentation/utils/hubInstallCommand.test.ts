import { describe, expect, it } from 'vitest'
import { parseHubInstallInput, tokenizeInstallCommand } from './hubInstallCommand'

describe('hubInstallCommand', () => {
  it('tokenizes quoted install commands', () => {
    expect(tokenizeInstallCommand('kageos install "user/app:0.1.0" --key "secret value"')).toEqual([
      'kageos',
      'install',
      'user/app:0.1.0',
      '--key',
      'secret value'
    ])
  })

  it('normalizes a Docker-like Hub reference with a version tag', () => {
    expect(parseHubInstallInput('kageos install user_1210227080/meeting_room_booking:0.1.0 --key abc')).toEqual({
      bundleUrl: 'https://hub-api.kageos.com/api/v1/applications/user_1210227080/meeting_room_booking/0.1.0/bundle',
      installKey: 'abc',
      displaySource: 'user_1210227080/meeting_room_booking:0.1.0'
    })
  })

  it('uses latest when the Docker-like reference omits a tag', () => {
    expect(parseHubInstallInput('user_1210227080/meeting_room_booking')).toEqual({
      bundleUrl: 'https://hub-api.kageos.com/api/v1/applications/user_1210227080/meeting_room_booking/latest/bundle',
      installKey: undefined,
      displaySource: 'user_1210227080/meeting_room_booking:latest'
    })
  })

  it('normalizes local registry references to http bundle URLs', () => {
    expect(parseHubInstallInput('kageos install localhost:4101/user_1210227080/meeting_room_booking:stable --key=abc')).toEqual({
      bundleUrl: 'http://localhost:4101/api/v1/applications/user_1210227080/meeting_room_booking/stable/bundle',
      installKey: 'abc',
      displaySource: 'user_1210227080/meeting_room_booking:stable'
    })
  })

  it('supports registry-qualified owners that contain email characters', () => {
    expect(parseHubInstallInput('api.kageos.com/alice@example.com/app:0.1.0')).toEqual({
      bundleUrl: 'https://api.kageos.com/api/v1/applications/alice%40example.com/app/0.1.0/bundle',
      installKey: undefined,
      displaySource: 'alice@example.com/app:0.1.0'
    })
  })

  it('keeps existing bundle URLs compatible', () => {
    expect(parseHubInstallInput('kageos install http://localhost:4101/api/v1/applications/user/app/0.1.0/bundle --key abc')).toEqual({
      bundleUrl: 'http://localhost:4101/api/v1/applications/user/app/0.1.0/bundle',
      installKey: 'abc',
      displaySource: 'http://localhost:4101/api/v1/applications/user/app/0.1.0/bundle'
    })
  })

  it('extracts query install keys while removing them from the bundle URL', () => {
    expect(parseHubInstallInput('https://hub.kageos.com/api/v1/applications/user/app/latest/bundle?key=abc')).toEqual({
      bundleUrl: 'https://hub.kageos.com/api/v1/applications/user/app/latest/bundle',
      installKey: 'abc',
      displaySource: 'https://hub.kageos.com/api/v1/applications/user/app/latest/bundle'
    })
  })

  it('parses bundle subpaths from install command flags', () => {
    expect(parseHubInstallInput('kageos install user/app:1.0.0 --key abc --path crm/customers')).toEqual({
      bundleUrl: 'https://hub-api.kageos.com/api/v1/applications/user/app/1.0.0/bundle',
      installKey: 'abc',
      bundleSubpath: 'crm/customers',
      displaySource: 'user/app:1.0.0#crm/customers'
    })
  })

  it('extracts URL subpaths while removing them from the bundle URL', () => {
    expect(parseHubInstallInput('https://hub.kageos.com/api/v1/applications/user/app/latest/bundle?key=abc&subpath=crm%2Fcustomers')).toEqual({
      bundleUrl: 'https://hub.kageos.com/api/v1/applications/user/app/latest/bundle',
      installKey: 'abc',
      bundleSubpath: 'crm/customers',
      displaySource: 'https://hub.kageos.com/api/v1/applications/user/app/latest/bundle#crm/customers'
    })
  })
})
