import { TestBed } from '@angular/core/testing';

import { LogDataSource } from './log.service';

describe('LogService', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('should be created', () => {
    const service: LogDataSource = TestBed.get(LogDataSource);
    expect(service).toBeTruthy();
  });
});
