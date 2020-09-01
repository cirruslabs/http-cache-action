import {createCachingServer} from "./server";
import * as request from "request";
import {expect} from "chai";

const server = createCachingServer();

describe('server response', function () {
    before(function () {
        server.listen(8000);
    });

    it('should return 404', function (done) {
        request
            .get('http://localhost:8000/foo')
            .on('response', function (response) {
                expect(response.statusCode).to.equal(404);
            })
    });

    after(function () {
        server.close();
    });
});