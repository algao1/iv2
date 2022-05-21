import gridfs
from pymongo import MongoClient


class Store:
    def __init__(self, uri) -> None:
        self.client = MongoClient(uri)

    def store_image(self, contents, filename):
        database = self.client["ichor"]
        fs = gridfs.GridFS(database)
        file = fs.find_one({"filename": filename})
        if file:
            return file._id
        return fs.put(contents, filename=filename)

    def retrieve_image(self, iid, filename=None):
        database = self.client["ichor"]
        fs = gridfs.GridFS(database)
        if not fs.exists(iid):
            raise FileNotFoundError(f"{iid} does not exist")
        return fs.get(iid)

